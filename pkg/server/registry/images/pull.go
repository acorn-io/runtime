package images

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImagePull(c kclient.WithWatch, clientFactory *client.Factory) *ImagePull {
	return &ImagePull{
		client:        c,
		clientFactory: clientFactory,
	}
}

type ImagePull struct {
	*strategy.DestroyAdapter
	client        kclient.WithWatch
	clientFactory *client.Factory
}

func (i *ImagePull) NamespaceScoped() bool {
	return true
}

func (i *ImagePull) New() runtime.Object {
	return &apiv1.LogOptions{}
}

func (i *ImagePull) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.LogOptions{}, false, ""
}

func (i *ImagePull) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	id = strings.ReplaceAll(id, "+", "/")
	ns, _ := request.NamespaceFrom(ctx)

	progress, err := i.ImagePull(ctx, ns, id)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		for update := range progress {
			p := ImageProgress{
				Total:    update.Total,
				Complete: update.Complete,
			}
			if update.Error != nil {
				p.Error = update.Error.Error()
			}
			data, err := json.Marshal(p)
			if err != nil {
				panic("failed to marshal update: " + err.Error())
			}
			_, _ = rw.Write(append(data, '\n'))
		}
	}), nil
}

func (i *ImagePull) ConnectMethods() []string {
	return []string{"POST"}
}

func (i *ImagePull) ImagePull(ctx context.Context, namespace, imageName string) (<-chan ggcrv1.Update, error) {
	writeOpts, err := remoteopts.Common(ctx)
	if err != nil {
		return nil, err
	}

	keyChain, err := pullsecret.Keychain(ctx, i.client, namespace)
	if err != nil {
		return nil, err
	}

	writeOpts = append(writeOpts, remote.WithAuthFromKeychain(keyChain))

	opts, err := remoteopts.WithServerDialer(ctx, i.client)
	if err != nil {
		return nil, err
	}

	pullTag, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err
	}

	index, err := remote.Index(pullTag, writeOpts...)
	if err != nil {
		return nil, err
	}

	hash, err := index.Digest()
	if err != nil {
		return nil, err
	}

	repo, err := getRepo(namespace)
	if err != nil {
		return nil, err
	}

	progress := make(chan ggcrv1.Update)
	// progress gets closed by remote.WriteIndex so this second channel is so that
	// we can control closing the result channel in case we need to write an error
	progress2 := make(chan ggcrv1.Update)
	opts = append(opts, remote.WithProgress(progress))
	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for update := range progress {
			progress2 <- update
		}
	}()

	go func() {
		defer func() {
			wg.Wait()
			close(progress2)
		}()

		// don't write error to chan because it already gets sent to the progress chan by remote.WriteIndex()
		if err = remote.WriteIndex(repo.Digest(hash.Hex), index, opts...); err == nil {
			if err := i.clientFactory.Namespace(namespace).ImageTag(ctx, hash.Hex, imageName); err != nil {
				progress2 <- ggcrv1.Update{
					Error: err,
				}
			}
		} else {
			handleWriteIndexError(err, progress)
		}
	}()

	return keepalive(progress2), nil
}

func keepalive(c <-chan ggcrv1.Update) <-chan ggcrv1.Update {
	result := make(chan ggcrv1.Update)
	go func() {
		var (
			lastUpdate ggcrv1.Update
			timer      = time.NewTicker(time.Second)
			ok         bool
		)
		defer close(result)
		defer timer.Stop()
		for {
			select {
			case lastUpdate, ok = <-c:
				if !ok {
					return
				}
			case <-timer.C:
			}
			result <- lastUpdate
		}
	}()
	return result
}
