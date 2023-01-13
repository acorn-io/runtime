package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewStop(c client.CommandContext) *cobra.Command {
	return cli.Command(&Stop{client: c.ClientFactory}, cobra.Command{
		Use: "stop [flags] [APP_NAME...]",
		Example: `
acorn stop my-app

acorn stop my-app1 my-app2`,
		SilenceUsage:      true,
		Short:             "Stop an app",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).complete,
	})
}

type Stop struct {
	client client.ClientFactory
}

func (a *Stop) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, arg := range args {
		err := c.AppStop(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("stopping %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}
