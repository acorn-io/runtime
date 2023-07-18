package cli

import (
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/credentials"
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
	Output string `usage:"Output format (json, yaml, aml)" short:"o" local:"true" default:"yaml"`
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

	cfg, err := a.client.Options().CLIConfig()
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

	type ImageDetailsOutput struct {
		internalv1.AppImage
		SignatureDigest string `json:"signature,omitempty" yaml:"signature,omitempty"`
	}

	w := table.NewWriter(nil, false, a.Output)
	w.WriteFormatted(ImageDetailsOutput{
		AppImage:        image.AppImage,
		SignatureDigest: image.SignatureDigest,
	}, nil)

	return w.Close()
}
