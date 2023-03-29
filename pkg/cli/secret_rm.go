package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewSecretDelete(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SecretDelete{client: c.ClientFactory}, cobra.Command{
		Use: "rm [SECRET_NAME...]",
		Example: `
acorn secret rm my-secret`,
		SilenceUsage:      true,
		Short:             "Delete a secret",
		ValidArgsFunction: newCompletion(c.ClientFactory, secretsCompletion).complete,
	})
	return cmd
}

type SecretDelete struct {
	client ClientFactory
}

func (a *SecretDelete) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, secret := range args {
		deleted, err := client.SecretDelete(cmd.Context(), secret)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", secret, err)
		}
		if deleted != nil {
			fmt.Println(secret)
		} else {
			fmt.Printf("Error: No such secret: %s\n", secret)
		}
	}

	return nil
}
