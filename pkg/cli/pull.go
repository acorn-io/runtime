package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/client"
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

	image, err := c.ImagePull(cmd.Context(), args[0])
	if err != nil {
		return err
	}
	fmt.Println(image.Digest)
	return nil
}
