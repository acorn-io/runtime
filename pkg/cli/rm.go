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
	Volumes    bool `usage:"Delete volumes" short:"v"`
	Images     bool `usage:"Delete images" short:"i"`
	Containers bool `usage:"Delete apps/containers" short:"c"`
	All        bool `usage:"Delete all types" short:"a"`
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	if a.All {
		a.Volumes = true
		a.Images = true
		a.Containers = true
	}

	for _, arg := range args {
		if a.Volumes {
			v, err := client.VolumeDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting volume %s: %w", arg, err)
			}
			if v != nil {
				fmt.Println(arg)
				continue
			}
		}

		if a.Images {
			i, err := client.ImageDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting image %s: %w", arg, err)
			}
			if i != nil {
				fmt.Println(arg)
				continue
			}
		}

		if a.Containers || (!a.Images && !a.Volumes) {
			app, err := client.AppDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting app %s: %w", arg, err)
			}
			if app != nil {
				fmt.Println(arg)
				continue
			}

			replica, err := client.ContainerReplicaDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting container %s: %w", arg, err)
			}
			if replica != nil {
				fmt.Println(arg)
				continue
			}
		}
	}

	return nil
}
