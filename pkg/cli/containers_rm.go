package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewContainerDelete(c CommandContext) *cobra.Command {
	cd := &ContainerDelete{client: c.ClientFactory}
	cmd := cli.Command(cd, cobra.Command{
		Use: "kill [CONTAINER_NAME...]",
		Example: `
acorn container kill app-name.containername-generated-hash`,
		SilenceUsage:      true,
		Short:             "Delete a container",
		Aliases:           []string{"rm", "delete"},
		ValidArgsFunction: newCompletion(c.ClientFactory, containersCompletion).complete,
	})
	return cmd
}

type ContainerDelete struct {
	client ClientFactory
}

func (a *ContainerDelete) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, container := range args {
		replicaDelete, err := c.ContainerReplicaDelete(cmd.Context(), container)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", container, err)
		}
		if replicaDelete != nil {
			fmt.Println(container)
		} else {
			fmt.Printf("Error: No such container: %s\n", container)
		}
	}

	return nil
}
