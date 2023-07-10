package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewPs(c CommandContext) *cobra.Command {
	return cli.Command(&Ps{client: c.ClientFactory}, cobra.Command{
		Use:     "ps [flags] [APP_NAME...]",
		Aliases: []string{"app", "apps", "a"},
		Example: `
acorn ps`,
		SilenceUsage:      true,
		Short:             "List or get apps",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).complete,
	})
}

type Ps struct {
	All         bool   `usage:"Include stopped apps" short:"a"`
	AllProjects bool   `usage:"Include all projects" short:"A"`
	Quiet       bool   `usage:"Output only names" short:"q"`
	Output      string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client      ClientFactory
}

func (a *Ps) Run(cmd *cobra.Command, args []string) error {
	var (
		c   client.Client
		err error
	)
	if a.AllProjects {
		c, err = a.client.CreateWithAllProjects()
	} else {
		c, err = a.client.CreateDefault()
	}
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.App, a.Quiet, a.Output)

	if len(args) == 1 {
		app, err := c.AppGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(app)
		return out.Err()
	}

	apps, err := c.AppList(cmd.Context())
	if err != nil {
		return err
	}

	for _, app := range apps {
		if (app.Status.AppStatus.Stopped || app.Status.AppStatus.Completed) && !a.All {
			continue
		}
		if len(args) > 0 {
			if slices.Contains(args, app.Name) {
				out.Write(&app)
			}
		} else {
			out.Write(&app)
		}
	}

	return out.Err()
}
