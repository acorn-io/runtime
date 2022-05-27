package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewRm() *cobra.Command {
	return cli.Command(&Rm{}, cobra.Command{
		Use: "rm [flags] [APP_NAME|VOL_NAME...]",
		Example: `
acorn rm
acorn rm -v some-volume`,
		SilenceUsage: true,
		Short:        "Delete an app, container, or volume",
	})
}

type Rm struct {
	Volumes    bool `usage:"Delete volumes" short:"v"`
	Images     bool `usage:"Delete images/tags" short:"i"`
	Secrets    bool `usage:"Delete secrets" short:"s"`
	Containers bool `usage:"Delete apps/containers" short:"c"`
	All        bool `usage:"Delete all types" short:"a"`
}

func (a *Rm) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	// If nothing is set default to containers
	if !(a.Images ||
		a.Volumes ||
		a.Secrets ||
		a.Containers) {
		a.Containers = true
	}

	if a.All {
		a.Volumes = true
		a.Images = true
		a.Containers = true
		a.Secrets = true
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

		if a.Containers {
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

		if a.Secrets {
			secret, err := client.SecretDelete(cmd.Context(), arg)
			if err != nil {
				return fmt.Errorf("deleting secret %s: %w", arg, err)
			}
			if secret != nil {
				fmt.Println(arg)
				continue
			}
		}
	}

	return nil
}
