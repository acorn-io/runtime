package cli

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewContainer(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Container{client: c.ClientFactory}, cobra.Command{
		Use:     "container [flags] [APP_NAME|CONTAINER_NAME...]",
		Aliases: []string{"containers", "c"},
		Example: `
acorn containers`,
		SilenceUsage:      true,
		Short:             "Manage containers",
		ValidArgsFunction: newCompletion(c.ClientFactory, containersCompletion).complete,
	})
	cmd.AddCommand(NewContainerDelete(c))
	return cmd
}

type Container struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	All    bool   `usage:"Include stopped containers" short:"a"`
	client ClientFactory
}

func (a *Container) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Container, a.Quiet, a.Output)

	switch len(args) {
	case 0:
		containers, err := c.ContainerReplicaList(cmd.Context(), nil)
		if err != nil {
			return err
		}
		printContainerReplicas(containers, a.All, &out)
	case 1:
		app, err := c.AppGet(cmd.Context(), args[0])
		if err != nil {
			// see if it's the name of a container instead
			container, err := c.ContainerReplicaGet(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out.Write(container)
		} else {
			containers, err := c.ContainerReplicaList(cmd.Context(), &client.ContainerReplicaListOptions{App: app.Name})
			if err != nil {
				return err
			}
			printContainerReplicas(containers, a.All, &out)
		}
	default:
		containers, err := c.ContainerReplicaList(cmd.Context(), nil)
		if err != nil {
			return err
		}
		for _, container := range containers {
			if slices.Contains(args, container.Name) {
				out.Write(container)
			}
		}
	}

	return out.Err()
}

func printContainerReplicas(cs []v1.ContainerReplica, all bool, out *table.Writer) {
	for _, c := range cs {
		if all || c.Status.Columns.State != "stopped" {
			(*out).Write(c)
		}
	}
}
