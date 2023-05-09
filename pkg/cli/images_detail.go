package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/credentials"
	"github.com/acorn-io/aml"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewImageDetails(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageDetails{client: c.ClientFactory}, cobra.Command{
		Use:               "details IMAGE_NAME [NESTED DIGEST]",
		Example:           `acorn image details my-image`,
		Aliases:           []string{"detail"},
		SilenceUsage:      true,
		Short:             "Show details of an Image",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).complete,
		Args:              cobra.MinimumNArgs(1),
	})
	return cmd
}

type ImageDetails struct {
	client ClientFactory
	Output string `usage:"Output format (json, yaml, aml)" short:"o" local:"true" default:"aml"`
}

func (a *ImageDetails) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	var nested string
	if len(args) > 1 {
		nested = args[1]
	}

	ref, err := name.ParseReference(args[0])
	if err != nil {
		return err
	}

	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	creds, err := credentials.NewStore(cfg, c)
	if err != nil {
		return err
	}

	auth, _, err := creds.Get(cmd.Context(), ref.Context().RegistryStr())
	if err != nil {
		return err
	}

	image, err := c.ImageDetails(cmd.Context(), args[0], &client.ImageDetailsOptions{
		NestedDigest: nested,
		Auth:         auth,
	})
	if err != nil {
		return err
	}

	switch a.Output {
	case "aml":
		out, err := aml.Marshal(image.AppImage)
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", out)
	}

	return nil
}
