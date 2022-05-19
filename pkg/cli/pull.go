package cli

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewPull() *cobra.Command {
	return cli.Command(&Pull{}, cobra.Command{
		Use:          "pull [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Pull an image to a remote registry",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Pull struct {
}

func (s *Pull) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	progress, err := c.ImagePull(cmd.Context(), args[0], nil)
	if err != nil {
		return err
	}

	return progressbar.Print(progress)
}
