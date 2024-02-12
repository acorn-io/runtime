package cli

import (
	"fmt"

	"github.com/acorn-io/runtime/pkg/version"
	"github.com/spf13/cobra"
)

func NewVersion() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "Version information for acorn",
		Example: "acorn version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("acorn version %s\n", version.Get().String())
		},
		Args: cobra.NoArgs,
	}

	return cmd
}
