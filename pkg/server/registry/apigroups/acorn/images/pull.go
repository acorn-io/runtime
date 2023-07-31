package images

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImagePull(c kclient.WithWatch, clientFactory *client.Factory, transport http.RoundTripper) *ImagePull {
	return &ImagePull{
		client:        c,
		clientFactory: clientFactory,
		transportOpt:  remote.WithTransport(transport),
	}
}

type ImagePull struct {
	*strategy.DestroyAdapter
	client        kclient.WithWatch
	clientFactory *client.Factory
	transportOpt  remote.Option
}

func (i *ImagePull) NamespaceScoped() bool {
	return true
}

func (i *ImagePull) New() runtime.Object {
	return &apiv1.ImagePull{}
}

func (i *ImagePull) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ImagePull{}, false, ""
}

func (i *ImagePull) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	id = strings.ReplaceAll(id, "+", "/")
	ns, _ := request.NamespaceFrom(ctx)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conn, err := k8schannel.Upgrader.Upgrade(rw, req, nil)
		if err != nil {
			logrus.Errorf("Error during handshake for image pull: %v", err)
			return
		}
		defer conn.Close()

		k8schannel.AddCloseHandler(conn)

		args := &apiv1.ImagePull{}
		if err := conn.ReadJSON(args); err != nil {
			_ = conn.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
			return
		}

		progress, err := i.ImagePull(ctx, ns, id, args.Auth)
		if err != nil {
			_ = conn.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
			return
		}

		for update := range progress {
			data, err := json.Marshal(update)
			if err != nil {
				panic("failed to marshal update: " + err.Error())
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				logrus.Errorf("Error writing pull status: %v", err)
				break
			}
		}

		_ = conn.CloseHandler()(websocket.CloseNormalClosure, "")
	}), nil
}

func (i *ImagePull) ConnectMethods() []string {
	return []string{"GET"}
}

func findSignatureImage(imageRef name.Reference, opts ...remote.Option) (name.Tag, ggcrv1.Image, error) {
	if digest, ok := imageRef.(name.Digest); ok {
		tag, hash, err := acornsign.FindSignature(digest, opts...)
		if err != nil {
			return name.Tag{}, nil, err
		}
		if hash.Hex == "" {
			return name.Tag{}, nil, nil
		}

		img, err := remote.Image(tag, opts...)

		return tag, img, err
	} else {
		digeststr, err := acornsign.SimpleDigest(imageRef, opts...)
		if err != nil {
			return name.Tag{}, nil, err
		}
		return findSignatureImage(imageRef.Context().Digest(digeststr), opts...)
	}
}

func (i *ImagePull) ImagePull(ctx context.Context, namespace, imageName string, auth *apiv1.RegistryAuth) (<-chan ImageProgress, error) {
	pullTag, err := imagesystem.ParseAndEnsureNotInternalRepo(ctx, i.client, namespace, imageName)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Pulling %s (%#v)", pullTag.String(), pullTag)

	opts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, pullTag.Context(), auth, i.client, namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	index, err := remote.Index(pullTag, opts...)
	if err != nil {
		return nil, err
	}

	hash, err := index.Digest()
	if err != nil {
		return nil, err
	}

	repo, externalRepo, err := imagesystem.GetInternalRepoForNamespace(ctx, i.client, namespace)
	if err != nil {
		return nil, err
	}

	recordRepo := ""
	if externalRepo {
		recordRepo = repo.String()
	}

	sigTag, sig, err := findSignatureImage(pullTag.Context().Digest(hash.String()), opts...)
	if err != nil {
		return nil, err
	}
	if sig != nil {
		sigHash, err := sig.Digest()
		if err != nil {
			return nil, err
		}
		logrus.Infof("Pulling signature %s for %s", sigHash.String(), pullTag.String())
		if err = remote.Write(repo.Tag(sigTag.TagStr()), sig, opts...); err != nil {
			logrus.Errorf("Error writing signature %s for image %s: %v", sigHash.String(), pullTag.String(), err)
			return nil, err
		}
	}

	type updates struct {
		updateChan chan ggcrv1.Update
		sourceName string
		destTag    name.Tag
	}

	// metachannel is used to send updates to another channel for each index to be copied
	metachannel := make(chan updates)

	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan ImageProgress)

	go func() {
		defer close(progress)
		for c := range metachannel {
			for update := range c.updateChan {
				var errString string
				if update.Error != nil {
					errString = update.Error.Error()
				}
				progress <- ImageProgress{
					Total:       update.Total,
					Complete:    update.Complete,
					Error:       errString,
					CurrentTask: fmt.Sprintf("Pulling %s %s ", c.sourceName, c.destTag.String()),
				}
			}
		}
	}()

	// Copy the image and signature
	go func() {
		defer close(metachannel)
		imageProgress := make(chan ggcrv1.Update)
		metachannel <- updates{
			updateChan: imageProgress,
			sourceName: "image",
			destTag:    pullTag.Context().Tag(pullTag.Identifier()),
		}

		// Don't write error to chan because it already gets sent to the currProgress chan by remote.WriteIndex().
		// remote.WriteIndex will also close the currProgress channel on its own.
		if err := remote.WriteIndex(repo.Digest(hash.Hex), index, append(opts, remote.WithProgress(imageProgress))...); err == nil {
			if err := i.recordImage(ctx, hash, namespace, imageName, recordRepo); err != nil {
				imageProgress <- ggcrv1.Update{
					Error: err,
				}
			}
		} else {
			handleWriteIndexError(err, imageProgress)
		}

		if sig != nil {
			signatureProgress := make(chan ggcrv1.Update)
			logrus.Infof("Pulling signature %s", sigTag.String())
			metachannel <- updates{
				updateChan: signatureProgress,
				sourceName: "signature",
				destTag:    sigTag,
			}
			if err := remote.Write(repo.Tag(sigTag.TagStr()), sig, append(opts, remote.WithProgress(signatureProgress))...); err != nil {
				handleWriteIndexError(err, signatureProgress)
			}
		}
	}()

	return typed.Every(500*time.Millisecond, progress), nil
}

func (i *ImagePull) recordImage(ctx context.Context, hash ggcrv1.Hash, namespace, imageName, recordRepo string) error {
	img := &v1.ImageInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hash.Hex,
			Namespace: namespace,
		},
		Repo:   recordRepo,
		Digest: hash.String(),
	}
	if err := i.client.Create(ctx, img); apierror.IsAlreadyExists(err) {
		if err := i.client.Get(ctx, router.Key(namespace, hash.Hex), img); err != nil {
			return err
		}
		img.Repo = recordRepo
		img.Remote = false
		if err := i.client.Update(ctx, img); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	return i.clientFactory.Namespace("", namespace).ImageTag(ctx, hash.Hex, imageName)
}
