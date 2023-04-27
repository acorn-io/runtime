package cli

import (
	"context"
	"os"
	"strings"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// RunAndHandleError will execute the command and then print the error
// message if one has occurred.  It will also call os.Exit with 0 or 1.
// This function never returns
func RunAndHandleError(ctx context.Context, cmd *cobra.Command) {
	cmd.SilenceErrors = true
	err := cmd.ExecuteContext(ctx)
	if err != nil {
		errString := err.Error()
		//If user uses --project/-j flag that does not exist k8s returns namespace error.
		//Replace namespace with project in error message to make it more user friendly.
		if cmd.Flag("project").Value.String() != "" {
			errString = strings.Replace(errString, "namespace", "project", 1)
		}
		pterm.Error.Println(errString)
		os.Exit(1)
	}
	os.Exit(0)
}
