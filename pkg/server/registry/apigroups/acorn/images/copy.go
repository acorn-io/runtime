package images

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageCopy(c kclient.WithWatch, transport http.RoundTripper) *ImageCopy {
	return &ImageCopy{
		client:       c,
		transportOpt: remote.WithTransport(transport),
	}
}

type ImageCopy struct {
	*strategy.DestroyAdapter
	client       kclient.WithWatch
	transportOpt remote.Option
}

func (i *ImageCopy) NamespaceScoped() bool {
	return true
}

func (i *ImageCopy) New() runtime.Object {
	return &apiv1.ImageCopy{}
}

func (i *ImageCopy) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ImageCopy{}, false, ""
}

func (i *ImageCopy) Connect(ctx context.Context, _ string, _ runtime.Object, _ rest.Responder) (http.Handler, error) {
	ns, _ := request.NamespaceFrom(ctx)

	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		conn, err := k8schannel.Upgrader.Upgrade(rw, req, nil)
		if err != nil {
			logrus.Errorf("Error during handshake for image pull: %v", err)
			return
		}
		defer conn.Close()

		k8schannel.AddCloseHandler(conn)

		args := &apiv1.ImageCopy{}
		if err := conn.ReadJSON(args); err != nil {
			_ = conn.CloseHandler()(websocket.CloseUnsupportedData, err.Error())
			return
		}

		if err := i.Validate(ctx, *args, ns); err != nil {
			_ = conn.CloseHandler()(websocket.CloseUnsupportedData, err.Error())
			return
		}

		var progress <-chan ImageProgress
		if !args.AllTags {
			progress, err = i.ImageCopy(ctx, ns, args.Source, args.Dest, args.SourceAuth, args.DestAuth, args.Force)
			if err != nil {
				_ = conn.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
				return
			}
		} else {
			progress, err = i.RepoCopy(ctx, ns, args.Source, args.Dest, args.SourceAuth, args.DestAuth, args.Force)
			if err != nil {
				_ = conn.CloseHandler()(websocket.CloseInternalServerErr, err.Error())
				return
			}
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

func (i *ImageCopy) ConnectMethods() []string {
	return []string{"GET"}
}

func (i *ImageCopy) Validate(ctx context.Context, args apiv1.ImageCopy, namespace string) error {
	if err := imagesystem.IsNotInternalRepo(ctx, i.client, namespace, args.Source); err != nil {
		return err
	}

	if err := imagesystem.IsNotInternalRepo(ctx, i.client, namespace, args.Dest); err != nil {
		return err
	}

	return nil
}

func (i *ImageCopy) ImageCopy(ctx context.Context, namespace, sourceImage, destImage string, srcAuth, dstAuth *apiv1.RegistryAuth, force bool) (<-chan ImageProgress, error) {
	// The source is allowed to be a local image, so we use getImageReference, which checks locally and remotely if it wasn't found.
	sourceRef, err := i.getImageReference(ctx, sourceImage, namespace)
	if err != nil {
		return nil, err
	}

	if sourceRef.Context().RegistryStr() == images.NoDefaultRegistry {
		return nil, fmt.Errorf("missing registry name (i.e. docker.io, ghcr.io) from source %s", sourceImage)
	}

	destRef, err := name.ParseReference(destImage, name.WithDefaultRegistry(images.NoDefaultRegistry))
	if err != nil {
		return nil, err
	}

	if destRef.Context().RegistryStr() == images.NoDefaultRegistry {
		return nil, fmt.Errorf("missing registry name (i.e. docker.io, ghcr.io) from destination %s", destImage)
	}

	sourceOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, sourceRef.Context(), srcAuth, i.client, namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	destOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, destRef.Context(), dstAuth, i.client, namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	sourceIndex, err := remote.Index(sourceRef, sourceOpts...)
	if err != nil {
		return nil, err
	}

	if !force {
		if err := errIfImageExistsAndIsDifferent(sourceIndex, destRef, destOpts); err != nil {
			return nil, err
		}
	}

	progress := make(chan ggcrv1.Update)
	// progress gets closed by remote.WriteIndex, so this second channel is so that
	// we can control closing the result channel in case we need to write an error.
	progress2 := make(chan ImageProgress)
	destOpts = append(destOpts, remote.WithProgress(progress))

	go func() {
		defer close(progress2)
		for update := range progress {
			var errString = ""
			if update.Error != nil {
				errString = update.Error.Error()
			}
			progress2 <- ImageProgress{
				Total:       update.Total,
				Complete:    update.Complete,
				Error:       errString,
				CurrentTask: fmt.Sprintf("Copying %s to %s", sourceImage, destImage),
			}
		}
	}()

	go func() {
		// Don't write error to chan because it already gets sent to the progress chan by remote.WriteIndex().
		// remote.WriteIndex will also close the currProgress channel on its own.
		if err := remote.WriteIndex(destRef, sourceIndex, destOpts...); err != nil {
			handleWriteIndexError(err, progress)
		}
	}()

	return typed.Every(500*time.Millisecond, progress2), nil
}

func (i *ImageCopy) RepoCopy(ctx context.Context, namespace, source, dest string, sourceAuth, destAuth *apiv1.RegistryAuth, force bool) (<-chan ImageProgress, error) {
	sourceRepo, err := name.NewRepository(source)
	if err != nil {
		return nil, err
	}

	destRepo, err := name.NewRepository(dest)
	if err != nil {
		return nil, err
	}

	sourceOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, sourceRepo.Registry, sourceAuth, i.client, namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	destOpts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, destRepo.Registry, destAuth, i.client, namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	sourceTags, err := remote.List(sourceRepo, sourceOpts...)
	if err != nil {
		return nil, err
	}

	sourceIndexes := make([]ggcrv1.ImageIndex, len(sourceTags))
	for i, tag := range sourceTags {
		sourceIndexes[i], err = remote.Index(sourceRepo.Tag(tag), sourceOpts...)
		if err != nil {
			return nil, err
		}
	}

	// Don't copy tags that already exist in the destination if force is not set
	if !force {
		var newIndexSlice []ggcrv1.ImageIndex
		for i, imageIndex := range sourceIndexes {
			if err := errIfImageExistsAndIsDifferent(imageIndex, destRepo.Tag(sourceTags[i]), destOpts); err == nil {
				newIndexSlice = append(newIndexSlice, imageIndex)
			}
		}
		sourceIndexes = newIndexSlice
	}

	type updates struct {
		updateChan chan ggcrv1.Update
		currentTag string
	}

	// metachannel is used to send another channel with updates for each tag to be copied
	metachannel := make(chan updates)
	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan ImageProgress)

	go func() {
		defer close(progress)
		for c := range metachannel {
			for update := range c.updateChan {
				var errString = ""
				if update.Error != nil {
					errString = update.Error.Error()
				}
				progress <- ImageProgress{
					Total:       update.Total,
					Complete:    update.Complete,
					Error:       errString,
					CurrentTask: fmt.Sprintf("Copying %s:%s to %s:%s", source, c.currentTag, dest, c.currentTag),
				}
			}
		}
	}()

	go func() {
		defer close(metachannel)
		for i, imageIndex := range sourceIndexes {
			currProgress := make(chan ggcrv1.Update)
			metachannel <- updates{
				updateChan: currProgress,
				currentTag: sourceTags[i],
			}

			// Don't write error to chan because it already gets sent to the currProgress chan by remote.WriteIndex().
			// remote.WriteIndex will also close the currProgress channel on its own.
			if err := remote.WriteIndex(destRepo.Tag(sourceTags[i]), imageIndex, append(destOpts, remote.WithProgress(currProgress))...); err != nil {
				handleWriteIndexError(err, currProgress)
			}
		}
	}()

	return typed.Every(500*time.Millisecond, progress), nil
}

func errIfImageExistsAndIsDifferent(sourceIndex ggcrv1.ImageIndex, destRef name.Reference, destOpts []remote.Option) error {
	// Make sure that we are not about to destroy the remote index
	destIndex, err := remote.Index(destRef, destOpts...)
	var terr *transport.Error
	if ok := errors.As(err, &terr); ok && terr.StatusCode == http.StatusNotFound {
		return nil
	} else if err != nil {
		return err
	}

	destDigest, err := destIndex.Digest()
	if err != nil {
		return err
	}

	sourceDigest, err := sourceIndex.Digest()
	if err != nil {
		return err
	}

	if destDigest.String() != sourceDigest.String() {
		return fmt.Errorf("not copying image to %s since it already exists", destRef.String())
	}

	return nil
}

// getImageReference returns a name.Reference for the given image.
// It checks the internal registry first.
func (i *ImageCopy) getImageReference(ctx context.Context, img, namespace string) (name.Reference, error) {
	safeName := strings.ReplaceAll(img, "/", "+")
	image := &apiv1.Image{}
	if err := i.client.Get(ctx, router.Key(namespace, safeName), image); err != nil {
		if apierrors.IsNotFound(err) {
			return name.ParseReference(img, name.WithDefaultRegistry(images.NoDefaultRegistry))
		}
		return nil, err
	}

	repo, _, err := imagesystem.GetInternalRepoForNamespace(ctx, i.client, image.Namespace)
	if err != nil {
		return nil, err
	}

	return name.ParseReference(repo.Digest(image.Digest).String())
}
