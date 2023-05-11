package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/version"
	"github.com/spf13/cobra"
)

func NewVersion(c CommandContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Version information for acorn",
		Example: "acorn version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("acorn version %s\n", version.Get().String())
		},
		Args: cobra.NoArgs,
	}

	return cmd
}
