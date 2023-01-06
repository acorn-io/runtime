package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/acorn-io/acorn/pkg/project"
	"github.com/pkg/errors"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// RunAndHandleError will execute the command and then print the error
// message if one has occurred.  It will also call os.Exit with 0 or 1.
// This function never returns
func RunAndHandleError(ctx context.Context, cmd *cobra.Command) {
	cmd.SilenceErrors = true
	err := cmd.ExecuteContext(ctx)
	if !pterm.RawOutput && errors.Is(err, project.ErrNoCurrentProject) {
		fmt.Println(project.NoProjectMessageNoHub)
		os.Exit(1)
	} else if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
