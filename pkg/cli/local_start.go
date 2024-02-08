package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/spf13/cobra"
)

func NewLocalStart() *cobra.Command {
	cmd := cli.Command(&LocalStart{}, cobra.Command{
		Use:          "start [flags]",
		Aliases:      []string{"delete"},
		SilenceUsage: true,
		Short:        "Start local development server",
	})
	return cmd
}

type LocalStart struct {
	Reset  bool `usage:"Delete existing server and all data before starting"`
	Delete bool `usage:"Delete existing server before starting"`
}

func (a *LocalStart) Run(cmd *cobra.Command, _ []string) (err error) {
	c, err := local.NewContainer()
	if err != nil {
		return err
	}

	if a.Reset {
		return c.Reset(cmd.Context(), true)
	} else if a.Delete {
		return c.Reset(cmd.Context(), false)
	}

	_, _, err = c.Upgrade(cmd.Context(), false)
	return err
}
