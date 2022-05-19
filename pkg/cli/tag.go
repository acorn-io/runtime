package cli

import (
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewTag() *cobra.Command {
	return cli.Command(&Tag{}, cobra.Command{
		Use:          "tag [flags] SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		SilenceUsage: true,
		Short:        "Tag an image",
		Args:         cobra.RangeArgs(2, 2),
	})
}

type Tag struct {
}

func (s *Tag) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	src, tag := args[0], args[1]

	return client.ImageTag(cmd.Context(), src, tag)
}
