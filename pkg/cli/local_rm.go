package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/spf13/cobra"
)

func NewLocalRm(c CommandContext) *cobra.Command {
	cmd := cli.Command(&LocalRm{}, cobra.Command{
		Use:          "rm [flags]",
		Aliases:      []string{"delete"},
		SilenceUsage: true,
		Short:        "Delete local development server",
	})
	return cmd
}

type LocalRm struct {
	State bool `usage:"Include associated state (acorns, secrets and volume data)"`
}

func (a *LocalRm) Run(cmd *cobra.Command, args []string) error {
	c, err := local.NewContainer(cmd.Context())
	if err != nil {
		return err
	}

	if err := c.Delete(cmd.Context(), a.State); err != nil {
		return err
	}
	fmt.Println("removed")
	return nil
}
