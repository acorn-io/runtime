package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewTag(c client.CommandContext) *cobra.Command {
	return cli.Command(&Tag{client: c.ClientFactory}, cobra.Command{
		Use:          "tag [flags] SOURCE_IMAGE[:TAG] TARGET_IMAGE[:TAG]",
		SilenceUsage: true,
		Short:        "Tag an image",
		Args:         cobra.RangeArgs(2, 2),
	})
}

type Tag struct {
	client client.ClientFactory
}

func (s *Tag) Run(cmd *cobra.Command, args []string) error {
	client, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	src, tag := args[0], args[1]

	return client.ImageTag(cmd.Context(), src, tag)
}
