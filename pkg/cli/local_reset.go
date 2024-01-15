package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/spf13/cobra"
)

func NewLocalReset(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Reset{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Reset local development server, deleting all data",
	})
	return cmd
}

type Reset struct {
}

func (a *Reset) Run(cmd *cobra.Command, args []string) error {
	c, err := local.NewContainer(cmd.Context())
	if err != nil {
		return err
	}

	if err := c.Reset(cmd.Context()); err != nil {
		return err
	}
	fmt.Println("running", system.DefaultImage())
	return nil
}
