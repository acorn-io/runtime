package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewVolumeDelete() *cobra.Command {
	cmd := cli.Command(&VolumeDelete{}, cobra.Command{
		Use: "rm [VOLUME_NAME...]",
		Example: `
acorn volume rm my-volume`,
		SilenceUsage: true,
		Short:        "Delete a volume",
	})
	return cmd
}

type VolumeDelete struct {
}

func (a *VolumeDelete) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, volume := range args {
		deleted, err := client.VolumeDelete(cmd.Context(), volume)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", volume, err)
		}
		if deleted != nil {
			fmt.Println(volume)
		} else {
			fmt.Printf("Error: No such volume: %s\n", volume)
		}
	}

	return nil
}
