package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewStart() *cobra.Command {
	return cli.Command(&Start{}, cobra.Command{
		Use: "start [flags] [APP_NAME...]",
		Example: `
acorn start my-app

acorn start my-app1 my-app2`,
		SilenceUsage: true,
		Short:        "Start an app",
	})
}

type Start struct {
}

func (a *Start) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, arg := range args {
		err := client.AppStart(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("starting %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}
