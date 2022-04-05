package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewRm() *cobra.Command {
	return cli.Command(&Rm{}, cobra.Command{
		Use: "rm [flags] [APP_NAME...]",
		Example: `
herd rm`,
		SilenceUsage: true,
		Short:        "Delete an app",
	})
}

type Rm struct {
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, arg := range args {
		err := client.AppDelete(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}