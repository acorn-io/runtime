package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewCredentialLogout() *cobra.Command {
	return cli.Command(&CredentialLogout{}, cobra.Command{
		Use:     "logout [flags] [SERVER_ADDRESS]",
		Aliases: []string{"rm"},
		Example: `
acorn logout ghcr.io`,
		SilenceUsage: true,
		Short:        "Remove registry credentials",
		Args:         cobra.ExactArgs(1),
	})
}

type CredentialLogout struct {
}

func (a *CredentialLogout) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	cred, err := client.CredentialDelete(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	if cred != nil {
		fmt.Println(cred.Name)
	}
	return nil
}
