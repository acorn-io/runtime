package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/edit"
	"github.com/spf13/cobra"
)

func NewEdit(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Edit{client: c.ClientFactory}, cobra.Command{
		Use:               "edit ACORN_NAME|SECRET_NAME",
		Example:           `acorn edit my-acorn`,
		SilenceUsage:      true,
		Short:             "Edits an acorn or secret",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, appsThenSecretsCompletion).complete,
	})
	return cmd
}

type Edit struct {
	client ClientFactory
}

func (a *Edit) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	return edit.Edit(cmd.Context(), c, args[0], false)
}
