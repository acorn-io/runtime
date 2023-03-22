package cli

import (
	"context"
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
		ValidArgsFunction: newCompletion(c.ClientFactory, containersCompletion).checkProjectPrefix().complete,
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
		// No app or container name supplied, list all containers
		if err := printContainerReplicas(cmd.Context(), c, nil, a.All, &out); err != nil {
			return err
		}
	case 1:
		// One app or container name supplied, only list matches
		app, err := c.AppGet(cmd.Context(), args[0])
		if err != nil {
			// See if it's the name of a container instead
			container, err := c.ContainerReplicaGet(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out.Write(container)
		} else {
			if err := printContainerReplicas(cmd.Context(), c, &client.ContainerReplicaListOptions{App: app.Name}, a.All, &out); err != nil {
				return err
			}
		}
	default:
		// More than one name supplied, iterate through containers and list any that match
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

func printContainerReplicas(ctx context.Context, c client.Client, opts *client.ContainerReplicaListOptions, all bool, out *table.Writer) error {
	cs, err := c.ContainerReplicaList(ctx, opts)
	if err != nil {
		return err
	}
	for _, c := range cs {
		if all || c.Status.Columns.State != "stopped" {
			(*out).Write(c)
		}
	}
	return nil
}
