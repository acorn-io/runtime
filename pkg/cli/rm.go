package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewRm() *cobra.Command {
	return cli.Command(&Rm{}, cobra.Command{
		Use: "rm [flags] [APP_NAME|VOL_NAME...]",
		Example: `
herd rm
herd rm -v some-volume`,
		SilenceUsage: true,
		Short:        "Delete an app, container, or volume",
	})
}

type Rm struct {
	Volumes bool `usage:"Delete volumes" short:"v"`
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, arg := range args {
		if a.Volumes {
			err := client.VolumeDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting volume %s: %w", arg, err)
			}
		}

		err := client.AppDelete(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("deleting app %s: %w", arg, err)
		}
		err = client.ContainerReplicaDelete(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("deleting container %s: %w", arg, err)
		}
		fmt.Println(arg)
	}

	return nil
}
