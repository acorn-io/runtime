package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/spf13/cobra"
)

func NewInit() *cobra.Command {
	return cli.Command(&Init{}, cobra.Command{
		Use: "init [flags]",
		Example: `
acorn init`,
		SilenceUsage: true,
		Short:        "Install and configure acorn in the cluster",
		Args:         cobra.NoArgs,
	})
}

type Init struct {
	Image  string `usage:"Override the default image used for the deployment"`
	Output string `usage:"Output manifests instead of applying them (json, yaml)" short:"o"`

	apiv1.Config

	Mode string `usage:"Initialize only 'config', 'resources', or 'both' (default 'both')"`
}

func (i *Init) Run(cmd *cobra.Command, args []string) error {
	var image = install.DefaultImage()
	if i.Image != "" {
		image = i.Image
	}

	return install.Install(cmd.Context(), image, &install.Options{
		OutputFormat: i.Output,
		Config:       i.Config,
	})
}
