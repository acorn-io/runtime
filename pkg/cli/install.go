package cli

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/spf13/cobra"
)

func NewInstall(c client.CommandContext) *cobra.Command {
	return cli.Command(&Install{client: c.ClientFactory}, cobra.Command{
		Use: "install [flags]",
		Example: `
acorn install`,
		Aliases:      []string{"init"},
		SilenceUsage: true,
		Short:        "Install and configure acorn in the cluster",
		Args:         cobra.NoArgs,
	})
}

type Install struct {
	SkipChecks bool `usage:"Bypass installation checks"`

	Image  string `usage:"Override the default image used for the deployment"`
	Output string `usage:"Output manifests instead of applying them (json, yaml)" short:"o"`

	APIServerReplicas  *int `usage:"acorn-api deployment replica count" name:"api-server-replicas"`
	ControllerReplicas *int `usage:"acorn-controller deployment replica count"`

	apiv1.Config
	client client.ClientFactory
}

func (i *Install) Run(cmd *cobra.Command, args []string) error {
	var image = install.DefaultImage()
	if i.Image != "" {
		image = i.Image
	}

	return install.Install(cmd.Context(), image, &install.Options{
		SkipChecks:         i.SkipChecks,
		OutputFormat:       i.Output,
		Config:             i.Config,
		APIServerReplicas:  i.APIServerReplicas,
		ControllerReplicas: i.ControllerReplicas,
	})
}
