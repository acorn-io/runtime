package cli

import (
	"github.com/acorn-io/acorn/pkg/install"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewInit() *cobra.Command {
	return cli.Command(&Init{}, cobra.Command{
		Use: "init [flags] [APP_NAME...]",
		Example: `
acorn init`,
		SilenceUsage: true,
		Short:        "Initial cluster for acorn use",
		Args:         cobra.NoArgs,
	})
}

type Init struct {
	Image  string `usage:"Override the default image used for the deployment"`
	Output string `usage:"Output manifests instead of applying them (json, yaml)" short:"o"`
}

func (i *Init) Run(cmd *cobra.Command, args []string) error {
	var image = install.DefaultImage()
	if i.Image != "" {
		image = i.Image
	}

	return install.Install(cmd.Context(), image, &install.Options{
		OutputFormat: i.Output,
	})
}
