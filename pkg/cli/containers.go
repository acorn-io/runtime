package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewContainer(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Container{client: c.ClientFactory}, cobra.Command{
		Use:     "container [flags] [APP_NAME...]",
		Aliases: []string{"containers", "c"},
		Example: `
acorn containers`,
		SilenceUsage: true,
		Short:        "Manage containers",
	})
	cmd.AddCommand(NewContainerDelete(c))
	return cmd
}

type Container struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	All    bool   `usage:"Include stopped containers" short:"a"`
	client client.ClientFactory
}

func (a *Container) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Container, system.UserNamespace(), a.Quiet, a.Output)

	if len(args) == 1 {
		app, err := client.ContainerReplicaGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(app)
		return out.Err()
	}

	containers, err := client.ContainerReplicaList(cmd.Context(), nil)
	if err != nil {
		return err
	}

	for _, container := range containers {
		if len(args) > 0 {
			if slices.Contains(args, container.Name) {
				out.Write(container)
			}
		} else if a.All || container.Status.Columns.State != "stopped" {
			out.Write(container)
		}
	}

	return out.Err()
}
