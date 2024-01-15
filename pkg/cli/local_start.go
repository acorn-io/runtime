package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/spf13/cobra"
)

func NewLocalStart(c CommandContext) *cobra.Command {
	cmd := cli.Command(&LocalStart{}, cobra.Command{
		Use:          "start [flags]",
		Aliases:      []string{"delete"},
		SilenceUsage: true,
		Short:        "Start local development server",
	})
	return cmd
}

type LocalStart struct {
}

func (a *LocalStart) Run(cmd *cobra.Command, args []string) error {
	c, err := local.NewContainer(cmd.Context())
	if err != nil {
		return err
	}

	if _, err := c.Create(cmd.Context(), false); err != nil {
		return err
	}

	if err := c.Start(cmd.Context()); err != nil {
		return err
	}

	fmt.Println("started")
	return nil
}
