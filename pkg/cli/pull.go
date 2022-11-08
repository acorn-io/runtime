package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/spf13/cobra"
)

func NewPull(c client.CommandContext) *cobra.Command {
	return cli.Command(&Pull{client: c.ClientFactory}, cobra.Command{
		Use:          "pull [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Pull an image from a remote registry",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Pull struct {
	client client.ClientFactory
}

func (s *Pull) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	progress, err := c.ImagePull(cmd.Context(), args[0], nil)
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}
