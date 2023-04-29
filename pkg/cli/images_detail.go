package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/aml"
	"github.com/spf13/cobra"
)

func NewImageDetail(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageDetail{client: c.ClientFactory}, cobra.Command{
		Use:               "detail IMAGE_NAME [NESTED DIGEST]",
		Example:           `acorn image detail my-image`,
		SilenceUsage:      true,
		Short:             "Show details of an Image",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).complete,
		Args:              cobra.MinimumNArgs(1),
	})
	return cmd
}

type ImageDetail struct {
	client ClientFactory
	Output string `usage:"Output format (json, yaml, aml)" short:"o" local:"true" default:"aml"`
}

func (a *ImageDetail) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	var nested string
	if len(args) > 1 {
		nested = args[1]
	}

	image, err := c.ImageDetails(cmd.Context(), args[0], &client.ImageDetailsOptions{
		NestedDigest: nested,
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
