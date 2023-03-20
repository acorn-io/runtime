package cli

import (
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewVolume(c CommandContext) *cobra.Command {
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
	client ClientFactory
}

func (a *Volume) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Volume, a.Quiet, a.Output)
	out.AddFormatFunc("alias", func(obj apiv1.Volume) string {
		return volumeAlias(&obj)
	})

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

// volumeAliases matches the correct app name to the given volume
func volumeAlias(volume *apiv1.Volume) string {

	if len(volume.Labels[labels.AcornVolumeName]) > 0 && len(volume.Labels[labels.AcornAppName]) > 0 {
		return fmt.Sprintf("%s.%s", volume.Labels[labels.AcornAppName], volume.Labels[labels.AcornVolumeName])
	}

	return ""
}
