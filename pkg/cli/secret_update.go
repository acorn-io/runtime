package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewSecretUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SecretUpdate{client: c.ClientFactory}, cobra.Command{
		Use: "update [flags] SECRET_NAME",
		Example: `
# Create secret with specific keys
acorn secret update --data key-name=value --data key-name2=value2 my-secret

# Read full secret from a file. The file should have a type and data field.
acorn secret update --file secret.yaml my-secret

# Read key value from a file
acorn secret update --data @key-name=secret.yaml my-secret`,
		SilenceUsage: true,
		Short:        "Update a secret",
		Args:         cobra.ExactArgs(1),
	})
	return cmd
}

type SecretUpdate struct {
	SecretFactory
	client ClientFactory
}

func (a *SecretUpdate) Run(cmd *cobra.Command, args []string) error {
	client, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	secret, err := a.buildSecret()
	if err != nil {
		return err
	}

	existing, err := client.SecretReveal(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	if existing.Data == nil {
		existing.Data = map[string][]byte{}
	}

	for k, v := range secret.Data {
		existing.Data[k] = v
	}

	newSecret, err := client.SecretUpdate(cmd.Context(), args[0], existing.Data)
	if err != nil {
		return err
	}

	fmt.Println(newSecret.Name)
	return nil
}
