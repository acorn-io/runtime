package cli

import (
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/client"
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

	var (
		nested string
		auth   *apiv1.RegistryAuth
	)

	if len(args) > 1 {
		nested = args[1]
	}

	auth, err = getAuthForImage(cmd.Context(), a.client, args[0])
	if err != nil {
		return err
	}

	image, err := c.ImageDetails(cmd.Context(), args[0], &client.ImageDetailsOptions{
		NestedDigest:  nested,
		Auth:          auth,
		IncludeNested: nested == "",
	})
	if err != nil {
		return err
	}

	w := table.NewWriter(nil, false, a.Output)
	w.WriteFormatted(image, nil)

	return w.Close()
}
