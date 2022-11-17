package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewSecretDelete() *cobra.Command {
	cmd := cli.Command(&SecretDelete{}, cobra.Command{
		Use: "rm [SECRET_NAME...]",
		Example: `
acorn secret rm my-secret`,
		SilenceUsage: true,
		Short:        "Delete a secret",
	})
	return cmd
}

type SecretDelete struct {
}

func (a *SecretDelete) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
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
