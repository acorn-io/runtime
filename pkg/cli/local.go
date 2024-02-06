package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewLocal() *cobra.Command {
	cmd := cli.Command(&Local{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Manage local development acorn runtime",
		Hidden:       true,
	})
	cmd.AddCommand(NewLocalServer())
	cmd.AddCommand(NewLocalLogs())
	cmd.AddCommand(NewLocalRm())
	cmd.AddCommand(NewLocalStart())
	cmd.AddCommand(NewLocalStop())
	return cmd
}

type Local struct {
}

func (a *Local) Run(*cobra.Command, []string) error {
	return nil
}
