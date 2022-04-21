package cli

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewApp() *cobra.Command {
	return cli.Command(&App{}, cobra.Command{
		Use:     "app [flags] [APP_NAME...]",
		Aliases: []string{"apps", "a", "ps"},
		Example: `
acorn app`,
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
		{"ENDPOINTS", "Status.Columns.Endpoints"},
		{"MESSAGE", "Status.Columns.Message"},
	}, "", a.Quiet, a.Output)

	if len(args) == 1 {
		app, err := client.AppGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(app)
		return out.Err()
	}

	apps, err := client.AppList(cmd.Context())
	if err != nil {
		return err
	}

	for _, app := range apps {
		if len(args) > 0 {
			if slices.Contains(args, app.Name) {
				out.Write(app)
			}
		} else {
			out.Write(app)
		}
	}

	return out.Err()
}
