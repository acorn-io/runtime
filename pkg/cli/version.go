package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/version"
	"github.com/spf13/cobra"
)

func NewVersion(c CommandContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "version",
		Short:   "version for acorn",
		Example: "acorn version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("acorn version %s\n", version.Get().String())
		},
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	}
	cmd.PersistentFlags().BoolP("verbose", "v", false, "Print verbose output")
	return cmd
}
