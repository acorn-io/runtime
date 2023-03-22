package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewStart(c CommandContext) *cobra.Command {
	return cli.Command(&Start{client: c.ClientFactory}, cobra.Command{
		Use: "start [flags] [APP_NAME...]",
		Example: `
acorn start my-app

acorn start my-app1 my-app2`,
		SilenceUsage:      true,
		Short:             "Start an app",
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).checkProjectPrefix().complete,
	})
}

type Start struct {
	client ClientFactory
}

func (a *Start) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, arg := range args {
		err := c.AppStart(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("starting %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}
