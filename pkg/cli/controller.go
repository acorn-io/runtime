package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/controller"
	"github.com/spf13/cobra"
)

func NewController(c CommandContext) *cobra.Command {
	return cli.Command(&Controller{client: c.ClientFactory}, cobra.Command{
		Use:          "controller",
		SilenceUsage: true,
		Hidden:       true,
		Short:        "Run k8s controller",
		Args:         cobra.NoArgs,
	})
}

type Controller struct {
	client ClientFactory
}

func (s *Controller) Run(cmd *cobra.Command, args []string) error {
	c, err := controller.New()
	if err != nil {
		return err
	}
	if err := c.Start(cmd.Context()); err != nil {
		return err
	}
	<-cmd.Context().Done()
	return nil
}
