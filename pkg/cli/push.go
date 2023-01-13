package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/spf13/cobra"
)

func NewPush(c client.CommandContext) *cobra.Command {
	return cli.Command(&Push{client: c.ClientFactory}, cobra.Command{
		Use:               "push [flags] IMAGE",
		SilenceUsage:      true,
		Short:             "Push an image to a remote registry",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(false)).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})
}

type Push struct {
	client client.ClientFactory
}

func (s *Push) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	prog, err := c.ImagePush(cmd.Context(), args[0], nil)
	if err != nil {
		return err
	}
	return progressbar.Print(prog)
}
