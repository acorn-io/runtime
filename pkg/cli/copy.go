package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/credentials"
	"github.com/acorn-io/runtime/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewImageCopy(c CommandContext) *cobra.Command {
	return cli.Command(&ImageCopy{client: c.ClientFactory}, cobra.Command{
		Use: `copy [flags] SOURCE DESTINATION

  This command can copy local images to remote registries, and can copy images between remote registries.
  It cannot copy images from remote registries to the local registry (use acorn pull instead).

  The --all-tags option only works with remote registries.`,
		Aliases:           []string{"cp"},
		SilenceUsage:      true,
		Short:             "Copy Acorn images between registries",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(false)).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `  # Copy the local image tagged "myimage:v1" to Docker Hub:
    acorn copy myimage:v1 docker.io/<username>/myimage:v1

  # Copy an image from Docker Hub to GHCR:
    acorn copy docker.io/<username>/myimage:v1 ghcr.io/<username>/myimage:v1

  # Copy all tags on a particular image repo in Docker Hub to GHCR:
    acorn copy --all-tags docker.io/<username>/myimage ghcr.io/<username>/myimage`,
	})
}

type ImageCopy struct {
	AllTags bool `usage:"Copy all tags of the image" short:"a"`
	Force   bool `usage:"Overwrite the destination image if it already exists" short:"f"`
	client  ClientFactory
}

func (a *ImageCopy) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	cfg, err := a.client.Options().CLIConfig()
	if err != nil {
		return err
	}

	creds, err := credentials.NewStore(cfg, c)
	if err != nil {
		return err
	}

	if !a.AllTags {
		// Check if the source argument matches the name of a local image, and use it if it does.
		// If there is an error, ignore it and move on, treating the argument as a remote image name.
		if _, err := c.ImageGet(cmd.Context(), args[0]); err == nil {
			return a.copyLocalToRemote(cmd, c, args, creds)
		}
	}

	source, err := name.ParseReference(args[0])
	if err != nil {
		return err
	}

	sourceAuth, _, err := creds.Get(cmd.Context(), source.Context().RegistryStr())
	if err != nil {
		return err
	}

	dest, err := name.ParseReference(args[1])
	if err != nil {
		return err
	}

	destAuth, _, err := creds.Get(cmd.Context(), dest.Context().RegistryStr())
	if err != nil {
		return err
	}

	progress, err := c.ImageCopy(cmd.Context(), args[0], args[1], &client.ImageCopyOptions{
		AllTags:    a.AllTags,
		Force:      a.Force,
		SourceAuth: sourceAuth,
		DestAuth:   destAuth,
	})
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}

func (a *ImageCopy) copyLocalToRemote(cmd *cobra.Command, c client.Client, args []string, creds *credentials.Store) error {
	dest, err := name.ParseReference(args[1])
	if err != nil {
		return err
	}

	destAuth, _, err := creds.Get(cmd.Context(), dest.Context().RegistryStr())
	if err != nil {
		return err
	}

	progress, err := c.ImageCopy(cmd.Context(), args[0], args[1], &client.ImageCopyOptions{
		Force:    a.Force,
		DestAuth: destAuth,
	})
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}
