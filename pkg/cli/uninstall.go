package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/uninstall"
	"github.com/spf13/cobra"
)

func NewUninstall(c client.CommandContext) *cobra.Command {
	return cli.Command(&Uninstall{client: c.ClientFactory}, cobra.Command{
		Use: "uninstall [flags]",
		Example: `
# Uninstall with confirmation
acorn uninstall

# Force uninstall without confirmation
acorn uninstall -f`,
		SilenceUsage: true,
		Short:        "Uninstall acorn and associated resources",
		Args:         cobra.NoArgs,
	})
}

type Uninstall struct {
	Force  bool `usage:"Do not prompt for confirmation" short:"f"`
	All    bool `usage:"Delete all volumes and secrets" short:"a"`
	client client.ClientFactory
}

func (u *Uninstall) Run(cmd *cobra.Command, args []string) error {
	return uninstall.Uninstall(cmd.Context(), &uninstall.Options{
		All:   u.All,
		Force: u.Force,
	})
}
