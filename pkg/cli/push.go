package cli

import (
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewPush() *cobra.Command {
	return cli.Command(&Push{}, cobra.Command{
		Use:          "push [flags] IMAGE",
		SilenceUsage: true,
		Short:        "Push an image to a remote registry",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Push struct {
}

func (s *Push) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}

	_, err = c.ImagePush(cmd.Context(), args[0])
	return err
}
