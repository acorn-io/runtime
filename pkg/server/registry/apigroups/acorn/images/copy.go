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
	"k8s.io/utils/strings/slices"
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
			logrus.Errorf("Error during handshake for image copy: %v", err)
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

	// Signature
	sigTag, sig, err := findSignatureImage(sourceRef.(name.Digest), sourceOpts...) // sourceRef is name.Digest as per getImageReference()
	if err != nil {
		return nil, err
	}
	sigPushTag := destRef.Context().Tag(sigTag.TagStr())

	// metachannel is used to send updates to another channel for each index to be copied
	metachannel := make(chan simpleUpdate)

	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan ImageProgress)

	go func() {
		defer close(progress)
		forwardUpdates(progress, metachannel)
	}()

	go func() {
		defer close(metachannel)
		remoteWrite(ctx, metachannel, destRef, sourceIndex, fmt.Sprintf("Copying %s to %s", sourceImage, destImage), nil, destOpts...)

		if sig != nil {
			remoteWrite(ctx, metachannel, sigPushTag, sig, fmt.Sprintf("Copying %s to %s", sigTag.String(), sigPushTag.String()), nil, destOpts...)
		}
	}()

	return typed.Every(500*time.Millisecond, progress), nil
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

	sourceTags, err := remote.List(sourceRepo, sourceOpts...) // remote.List will already include signature tags
	if err != nil {
		return nil, err
	}

	var sourceIndexes []ggcrv1.ImageIndex
	var sourceImages []ggcrv1.Image
	var sourceImagesTags []string
	for _, tag := range sourceTags {
		x, err := remote.Head(sourceRepo.Tag(tag), sourceOpts...)
		if err != nil {
			return nil, err
		}

		if x.MediaType.IsIndex() {
			index, err := remote.Index(sourceRepo.Tag(tag), sourceOpts...)
			if err != nil {
				return nil, err
			}
			sourceIndexes = append(sourceIndexes, index)
		} else if x.MediaType.IsImage() {
			img, err := remote.Image(sourceRepo.Tag(tag), sourceOpts...)
			if err != nil {
				return nil, err
			}
			sourceImages = append(sourceImages, img)
			sourceImagesTags = append(sourceImagesTags, tag)
		} else {
			return nil, fmt.Errorf("unknown media type for tag %s", tag)
		}
	}

	// sourceTags should only be ImageIndex tags at this point
	sourceTags = slices.Filter(nil, sourceTags, func(tag string) bool { return !slices.Contains(sourceImagesTags, tag) })

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

	// metachannel is used to send another channel with updates for each tag to be copied
	metachannel := make(chan simpleUpdate)
	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan ImageProgress)

	go func() {
		defer close(progress)
		forwardUpdates(progress, metachannel)
	}()

	go func() {
		defer close(metachannel)
		for i, imageIndex := range sourceIndexes {
			remoteWrite(ctx, metachannel, destRepo.Tag(sourceTags[i]), imageIndex, fmt.Sprintf("Copying index %s:%s to %s:%s", source, sourceTags[i], dest, sourceTags[i]), nil, destOpts...)
		}

		for i, img := range sourceImages {
			remoteWrite(ctx, metachannel, destRepo.Tag(sourceImagesTags[i]), img, fmt.Sprintf("Copying image %s:%s to %s:%s", source, sourceImagesTags[i], dest, sourceImagesTags[i]), nil, destOpts...)
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
