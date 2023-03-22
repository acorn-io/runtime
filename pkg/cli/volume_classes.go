package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewVolumeClasses(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Storage{client: c.ClientFactory}, cobra.Command{
		Use:     "volumeclasses [flags] [VOLUME_CLASS...]",
		Aliases: []string{"volumeclass", "vc"},
		Example: `
acorn offering volumeclasses`,
		SilenceUsage:      true,
		Short:             "List available volume classes",
		ValidArgsFunction: newCompletion(c.ClientFactory, volumeClassCompletion).checkProjectPrefix().complete,
	})
	return cmd
}

type Storage struct {
	Quiet  bool `usage:"Output only names" short:"q"`
	client ClientFactory
}

func (a *Storage) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.VolumeClass, a.Quiet, "")

	if len(args) == 1 {
		volume, err := c.VolumeClassGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(volume)
		return out.Err()
	}

	storages, err := c.VolumeClassList(cmd.Context())
	if err != nil {
		return err
	}

	for _, storage := range storages {
		if len(args) > 0 {
			if slices.Contains(args, storage.Name) {
				out.Write(storage)
			}
		} else {
			out.Write(storage)
		}
	}

	return out.Err()
}
