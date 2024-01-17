package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewLocal(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Local{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Manage local development acorn runtime",
		Hidden:       true,
	})
	cmd.AddCommand(NewLocalServer(c))
	cmd.AddCommand(NewLocalLogs(c))
	cmd.AddCommand(NewLocalRm(c))
	cmd.AddCommand(NewLocalStart(c))
	cmd.AddCommand(NewLocalStop(c))
	return cmd
}

type Local struct {
}

func (a *Local) Run(cmd *cobra.Command, args []string) error {
	return nil
}
