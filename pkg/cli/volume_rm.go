package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewVolumeDelete(c CommandContext) *cobra.Command {
	cmd := cli.Command(&VolumeDelete{client: c.ClientFactory}, cobra.Command{
		Use:               "rm [VOLUME_NAME...]",
		Example:           `acorn volume rm my-volume`,
		SilenceUsage:      true,
		Short:             "Delete a volume",
		ValidArgsFunction: newCompletion(c.ClientFactory, volumesCompletion).complete,
	})
	return cmd
}

type VolumeDelete struct {
	client ClientFactory
}

func (a *VolumeDelete) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, volume := range args {
		deleted, err := c.VolumeDelete(cmd.Context(), volume)
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
