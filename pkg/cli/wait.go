package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/wait"
	"github.com/spf13/cobra"
)

func NewWait(c CommandContext) *cobra.Command {
	return cli.Command(&Wait{client: c.ClientFactory}, cobra.Command{
		Use:               "wait [flags] APP_NAME",
		SilenceUsage:      true,
		Short:             "Wait an app to be ready then exit with status code 0",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, appsCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
}

type Wait struct {
	Quiet  bool `usage:"Do not print status" short:"q"`
	client ClientFactory
}

func (w *Wait) Run(cmd *cobra.Command, args []string) error {
	c, err := w.client.CreateDefault()
	if err != nil {
		return err
	}

	return wait.App(cmd.Context(), c, args[0], w.Quiet)
}
