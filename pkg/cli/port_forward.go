package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/portforward"
	"github.com/spf13/cobra"
)

func NewPortForward(c CommandContext) *cobra.Command {
	exec := &PortForward{client: c.ClientFactory}
	cmd := cli.Command(exec, cobra.Command{
		Use:               "port-forward [flags] APP_NAME|CONTAINER_NAME PORT",
		SilenceUsage:      true,
		Short:             "Forward a container port locally",
		Long:              "Forward a container port locally",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsThenContainersCompletion).complete,
		Args:              cobra.ExactArgs(2),
	})

	// This will produce an error if the container flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("container", newCompletion(c.ClientFactory, acornContainerCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for -c flag: %v\n", err)
	}

	return cmd
}

type PortForward struct {
	Container string `usage:"Name of container to port forward into" short:"c"`
	Address   string `usage:"The IP address to listen on" default:"127.0.0.1"`
	client    ClientFactory
}

func (s *PortForward) Run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	name, portDef := args[0], args[1]
	if err != nil {
		return err
	}

	app, appErr := c.AppGet(ctx, name)
	if appErr == nil {
		name, err = getContainerForApp(ctx, c, app, s.Container, true)
		if err != nil {
			return err
		}
	}
	return portforward.PortForward(ctx, c, name, s.Address, portDef)
}
