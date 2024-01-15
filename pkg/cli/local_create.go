package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/spf13/cobra"
)

func NewLocalCreate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Create{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Create local development server",
	})
	return cmd
}

type Create struct {
	Upgrade bool `usage:"Upgrade if runtime already exists"`
}

func (a *Create) Run(cmd *cobra.Command, args []string) error {
	c, err := local.NewContainer(cmd.Context())
	if err != nil {
		return err
	}

	if _, err := c.Create(cmd.Context(), a.Upgrade); err != nil {
		return err
	}
	fmt.Println("running", system.DefaultImage())
	return nil
}
