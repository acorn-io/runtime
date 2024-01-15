package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/spf13/cobra"
)

func NewLocalStop(c CommandContext) *cobra.Command {
	cmd := cli.Command(&LocalStop{}, cobra.Command{
		Use:          "stop [flags]",
		Aliases:      []string{"delete"},
		SilenceUsage: true,
		Short:        "Stop local development server",
	})
	return cmd
}

type LocalStop struct {
}

func (a *LocalStop) Run(cmd *cobra.Command, args []string) error {
	c, err := local.NewContainer(cmd.Context())
	if err != nil {
		return err
	}

	if err := c.Stop(cmd.Context()); err != nil {
		return err
	}
	fmt.Println("stopped")
	return nil
}
