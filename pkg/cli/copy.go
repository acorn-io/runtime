package cli

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/acorn-io/baaah/pkg/typed"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	ggcrv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewImageCopy(c CommandContext) *cobra.Command {
	return cli.Command(&ImageCopy{client: c.ClientFactory}, cobra.Command{
		Use: `copy [flags] SOURCE DESTINATION

  This command copies Acorn images between remote image registries.
  It does not interact with images stored in the Acorn internal registry, or with the Acorn API in any way.
  To set up credentials for a registry, use 'acorn login -l <registry>'. It only works with locally stored credentials.`,
		Aliases:           []string{"cp"},
		SilenceUsage:      true,
		Short:             "Copy Acorn images between registries",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(false)).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `  # Copy an image from Docker Hub to GHCR:
    acorn copy docker.io/<username>/myimage:v1 ghcr.io/<username>/myimage:v1

  # Copy the 'main' tag on an image to the 'prod' tag on the same image, and overwrite if it already exists:
    acorn copy docker.io/<username>/myimage:main prod --force

  # Copy all tags on a particular image repo in Docker Hub to GHCR:
    acorn copy --all-tags docker.io/<username>/myimage ghcr.io/<username>/myimage`,
	})
}

type ImageCopy struct {
	AllTags bool `usage:"Copy all tags of the image" short:"a"`
	Force   bool `usage:"Overwrite the destination image if it already exists" short:"f"`
	client  ClientFactory
}

func (a *ImageCopy) Run(cmd *cobra.Command, args []string) (err error) {
	// Print a helpful error message for the user if they end up getting an authentication error
	defer func() {
		if err != nil {
			var terr *transport.Error
			if ok := errors.As(err, &terr); ok {
				if terr.StatusCode == http.StatusForbidden {
					logrus.Warnf("Registry authentication failed. Try running 'acorn login -l <registry>'")
				} else if terr.StatusCode == http.StatusUnauthorized {
					logrus.Warnf("Registry authorization failed. Ensure that you have the correct permissions to push to this registry. Run 'acorn login -l <registry>' if you have not logged in yet.")
				}
			}
		}
	}()

	source, err := name.ParseReference(args[0], name.WithDefaultRegistry(images.NoDefaultRegistry))
	if err != nil {
		return err
	}
	if source.Context().RegistryStr() == images.NoDefaultRegistry {
		return fmt.Errorf("image %s has no specified registry", args[0])
	}

	sourceAuth, err := getAuthForImage(cmd.Context(), a.client, args[0])
	if err != nil {
		return err
	}
	sourceOpts := []remote.Option{remote.WithContext(cmd.Context())}
	if sourceAuth != nil {
		sourceKeychain := images.NewSimpleKeychain(source.Context(), *sourceAuth, nil)
		sourceOpts = append(sourceOpts, remote.WithAuthFromKeychain(sourceKeychain))
	}

	dest, err := name.ParseReference(args[1], name.WithDefaultRegistry(images.NoDefaultRegistry))
	if err != nil {
		return err
	}
	if dest.Context().RegistryStr() == images.NoDefaultRegistry {
		// If the dest has no registry, then it is just a new tag to create on the source image
		return a.copyTag(source, args[1], sourceOpts)
	}

	destAuth, err := getAuthForImage(cmd.Context(), a.client, args[1])
	if err != nil {
		return err
	}
	destOpts := []remote.Option{remote.WithContext(cmd.Context())}
	if destAuth != nil {
		destKeychain := images.NewSimpleKeychain(dest.Context(), *destAuth, nil)
		destOpts = append(destOpts, remote.WithAuthFromKeychain(destKeychain))
	}

	if a.AllTags {
		return a.copyRepo(args, sourceOpts, destOpts)
	}

	sourceIndex, err := remote.Index(source, sourceOpts...)
	if err != nil {
		return err
	}

	if !a.Force {
		if err := errIfImageExistsAndIsDifferent(sourceIndex, dest, destOpts); err != nil {
			return err
		}
	}

	// Signature
	sourceDigest, err := acornsign.SimpleDigest(source, sourceOpts...)
	if err != nil {
		return err
	}
	sigTag, sig, err := acornsign.FindSignatureImage(source.Context().Digest(sourceDigest), sourceOpts...)
	if err != nil {
		return err
	}
	sigPushTag := dest.Context().Tag(sigTag.TagStr())

	// metachannel is used to send another channel with updates for each tag to be copied
	metachannel := make(chan images.SimpleUpdate)
	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan images.ImageProgress)

	go func() {
		defer close(progress)
		images.ForwardUpdates(progress, metachannel)
	}()

	go func() {
		defer close(metachannel)
		images.RemoteWrite(metachannel, dest, sourceIndex, fmt.Sprintf("Copying %s to %s", args[0], args[1]), nil, destOpts...)

		if sig != nil {
			images.RemoteWrite(metachannel, sigPushTag, sig, fmt.Sprintf("Copying %s to %s", sigTag.String(), sigPushTag.String()), nil, destOpts...)
		}
	}()

	return progressbar.Print(adaptChannel(progress))
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
		return fmt.Errorf("not copying image to %s since it already exists (use --force to override)", destRef.String())
	}
	return nil
}

func (a *ImageCopy) copyRepo(args []string, sourceOpts, destOpts []remote.Option) error {
	sourceRepo, err := name.NewRepository(args[0], name.WithDefaultRegistry(images.NoDefaultRegistry))
	if err != nil {
		return err
	}
	if sourceRepo.RegistryStr() == images.NoDefaultRegistry {
		return fmt.Errorf("repo %s has no specified registry", args[0])
	}

	destRepo, err := name.NewRepository(args[1], name.WithDefaultRegistry(images.NoDefaultRegistry))
	if err != nil {
		return err
	}
	if destRepo.RegistryStr() == images.NoDefaultRegistry {
		return fmt.Errorf("repo %s has no specified registry", args[1])
	}

	sourceTags, err := remote.List(sourceRepo, sourceOpts...) // remote.List will already include signature tags
	if err != nil {
		return err
	}

	var sourceIndexes []ggcrv1.ImageIndex
	var sourceImages []ggcrv1.Image
	var sourceImagesTags []string
	for _, tag := range sourceTags {
		x, err := remote.Head(sourceRepo.Tag(tag), sourceOpts...)
		if err != nil {
			return err
		}

		if x.MediaType.IsIndex() {
			index, err := remote.Index(sourceRepo.Tag(tag), sourceOpts...)
			if err != nil {
				return err
			}
			sourceIndexes = append(sourceIndexes, index)
		} else if x.MediaType.IsImage() {
			img, err := remote.Image(sourceRepo.Tag(tag), sourceOpts...)
			if err != nil {
				return err
			}
			sourceImages = append(sourceImages, img)
			sourceImagesTags = append(sourceImagesTags, tag)
		} else {
			return fmt.Errorf("unknown media type for tag %s", tag)
		}
	}

	// sourceTags should only be ImageIndex tags at this point
	sourceTags = slices.Filter(nil, sourceTags, func(tag string) bool { return !slices.Contains(sourceImagesTags, tag) })

	// Don't copy tags that already exist in the destination if force is not set
	if !a.Force {
		var newIndexSlice []ggcrv1.ImageIndex
		for i, imageIndex := range sourceIndexes {
			if err := errIfImageExistsAndIsDifferent(imageIndex, destRepo.Tag(sourceTags[i]), destOpts); err == nil {
				newIndexSlice = append(newIndexSlice, imageIndex)
			}
		}
		sourceIndexes = newIndexSlice
	}

	// metachannel is used to send another channel with updates for each tag to be copied
	metachannel := make(chan images.SimpleUpdate)
	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan images.ImageProgress)

	go func() {
		defer close(progress)
		images.ForwardUpdates(progress, metachannel)
	}()

	go func() {
		defer close(metachannel)
		for i, imageIndex := range sourceIndexes {
			images.RemoteWrite(metachannel, destRepo.Tag(sourceTags[i]), imageIndex, fmt.Sprintf("Copying index %s:%s to %s:%s", args[0], sourceTags[i], args[1], sourceTags[i]), nil, destOpts...)
		}

		for i, img := range sourceImages {
			images.RemoteWrite(metachannel, destRepo.Tag(sourceImagesTags[i]), img, fmt.Sprintf("Copying image %s:%s to %s:%s", args[0], sourceImagesTags[i], args[1], sourceImagesTags[i]), nil, destOpts...)
		}
	}()

	return progressbar.Print(adaptChannel(progress))
}

func (a *ImageCopy) copyTag(source name.Reference, newTag string, sourceOpts []remote.Option) error {
	// -a is not supported for this operation, so check if it is set and return an error if so
	if a.AllTags {
		return errors.New("cannot use --all-tags with a tag destination")
	}

	sourceIndex, err := remote.Index(source, sourceOpts...)
	if err != nil {
		return err
	}

	dest := source.Context().Tag(newTag)

	// Parse it again to make sure that the tag provided by the user is valid
	_, err = name.ParseReference(dest.String())
	if err != nil {
		return err
	}

	if !a.Force {
		if err := errIfImageExistsAndIsDifferent(sourceIndex, dest, sourceOpts); err != nil {
			return err
		}
	}

	// No need to copy signature, since we are just creating a new tag in the same repository

	// metachannel is used to send another channel with updates for each tag to be copied
	metachannel := make(chan images.SimpleUpdate)
	// progress is the channel returned by this function and used to write websocket messages to the client
	progress := make(chan images.ImageProgress)

	go func() {
		defer close(progress)
		images.ForwardUpdates(progress, metachannel)
	}()

	go func() {
		defer close(metachannel)
		images.RemoteWrite(metachannel, dest, sourceIndex, fmt.Sprintf("Copying %s to %s", source.String(), newTag), nil, sourceOpts...)
	}()

	return progressbar.Print(adaptChannel(progress))
}

func adaptChannel(progress chan images.ImageProgress) <-chan client.ImageProgress {
	clientProgress := make(chan client.ImageProgress)
	go func() {
		defer close(clientProgress)
		for p := range progress {
			clientProgress <- client.ImageProgress{
				Total:       p.Total,
				Complete:    p.Complete,
				Error:       p.Error,
				CurrentTask: p.CurrentTask,
			}
		}
	}()
	return typed.Every(500*time.Millisecond, clientProgress)
}
