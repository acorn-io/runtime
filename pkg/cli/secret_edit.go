package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/edit"
	"github.com/spf13/cobra"
)

func NewSecretEdit(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SecretEdit{client: c.ClientFactory}, cobra.Command{
		Use:               "edit SECRET_NAME",
		Example:           `acorn secret edit my-secret`,
		SilenceUsage:      true,
		Short:             "Edits a secret",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, secretsCompletion).complete,
	})
	return cmd
}

type SecretEdit struct {
	client ClientFactory
}

func (a *SecretEdit) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	return edit.Edit(cmd.Context(), c, args[0], true)
}
