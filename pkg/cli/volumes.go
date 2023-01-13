package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewVolume(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Volume{client: c.ClientFactory}, cobra.Command{
		Use:     "volume [flags] [VOLUME_NAME...]",
		Aliases: []string{"volumes", "v"},
		Example: `
acorn volume`,
		SilenceUsage:      true,
		Short:             "Manage volumes",
		ValidArgsFunction: newCompletion(c.ClientFactory, volumesCompletion).complete,
	})
	cmd.AddCommand(NewVolumeDelete(c))
	return cmd
}

type Volume struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client client.ClientFactory
}

func (a *Volume) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Volume, system.UserNamespace(), a.Quiet, a.Output)

	if len(args) == 1 {
		volume, err := c.VolumeGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(volume)
		return out.Err()
	}

	volumes, err := c.VolumeList(cmd.Context())
	if err != nil {
		return err
	}

	for _, volume := range volumes {
		if len(args) > 0 {
			if slices.Contains(args, volume.Name) {
				out.Write(volume)
			}
		} else {
			out.Write(volume)
		}
	}

	return out.Err()
}
