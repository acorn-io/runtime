package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewCredentialLogout(root bool, c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&CredentialLogout{client: c.ClientFactory}, cobra.Command{
		Use:     "logout [flags] [SERVER_ADDRESS]",
		Aliases: []string{"rm"},
		Example: `
acorn logout ghcr.io`,
		SilenceUsage:      true,
		Short:             "Remove registry credentials",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, credentialsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
	if root {
		cmd.Aliases = nil
	}
	return cmd
}

type CredentialLogout struct {
	client client.ClientFactory
}

func (a *CredentialLogout) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
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
