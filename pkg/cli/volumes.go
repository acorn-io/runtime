package cli

import (
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler-cli/pkg/table"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewVolume() *cobra.Command {
	cmd := cli.Command(&Volume{}, cobra.Command{
		Use:     "volume [flags] [VOLUME_NAME...]",
		Aliases: []string{"volumes", "v"},
		Example: `
herd volume`,
		SilenceUsage: true,
		Short:        "List or get volumes",
	})
	cmd.AddCommand(NewVolume())
	return cmd
}

type Volume struct {
	Quiet  bool   `desc:"Output only names" short:"q"`
	Output string `desc:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *Volume) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	out := table.NewWriter([][]string{
		{"NAME", "Name"},
		{"APPNAME", "Status.AppName"},
		{"BOUNDVOLUME", "Status.VolumeName"},
		{"CAPACITY", "Capacity"},
		{"STATUS", "Status.Status"},
		{"ACCESSMODES", "Status.Columns.AccessModes"},
		{"CREATED", "{{ago .Created}}"},
		{"MESSAGE", "Status.Message"},
	}, "", a.Quiet, a.Output)

	if len(args) == 1 {
		volume, err := client.VolumeGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(volume)
		return out.Err()
	}

	volumes, err := client.VolumeList(cmd.Context())
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
