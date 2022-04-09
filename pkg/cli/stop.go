package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewStop() *cobra.Command {
	return cli.Command(&Stop{}, cobra.Command{
		Use: "stop [flags] [APP_NAME...]",
		Example: `
herd stop my-app

herd stop my-app1 my-app2`,
		SilenceUsage: true,
		Short:        "Stop an app",
	})
}

type Stop struct {
}

func (a *Stop) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, arg := range args {
		err := client.AppStop(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("stopping %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}
