package cli

import (
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
)

func NewApp() *cobra.Command {
	return cli.Command(&App{}, cobra.Command{
		Use:     "app [flags] [APP_NAME...]",
		Aliases: []string{"apps", "a"},
		Example: `
herd app`,
		SilenceUsage: true,
		Short:        "List or get apps",
	})
}

type App struct {
	Quiet  bool   `desc:"Output only names" short:"q"`
	Output string `desc:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *App) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter([][]string{
		{"NAME", "Name"},
		{"IMAGE", "Image"},
		{"HEALTHY", "Status.Columns.Healthy"},
		{"UPTODATE", "Status.Columns.UpToDate"},
		{"CREATED", "{{ago .Created}}"},
		{"MESSAGE", "Status.Columns.Message"},
	}, "", a.Quiet, a.Output)

	apps, err := client.AppList(cmd.Context())
	if err != nil {
		return err
	}

	for _, app := range apps {
		out.Write(app)
	}

	return out.Err()
}
