package cli

import (
	"fmt"

	hclient "github.com/ibuildthecloud/herd/pkg/client"
	cli "github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
)

func NewVolumeCreate() *cobra.Command {
	return cli.Command(&VolumeCreate{}, cobra.Command{
		Use:     "volume create [flags] VOLUME_NAME CAPACITY",
		Aliases: []string{"volumes", "v"},
		Example: `
herd volume create my-vol 10G`,
		SilenceUsage: true,
		Short:        "List or get volumes",
		Args:         cobra.ExactArgs(2),
	})
}

type VolumeCreate struct {
	Class      string   `usage:"Storage class, values are environment specific"`
	AccessMode []string `usage:"Access modes rwo/rwx/rox/rwop"`
}

func (a *VolumeCreate) Run(cmd *cobra.Command, args []string) error {
	c, err := hclient.Default()
	if err != nil {
		return err
	}

	name := args[0]
	quantity, err := resource.ParseQuantity(args[1])
	if err != nil {
		return err
	}

	accessModes, err := hclient.ToAccessModes(a.AccessMode)
	if err != nil {
		return err
	}

	vol, err := c.VolumeCreate(cmd.Context(), name, quantity, &hclient.VolumeCreateOptions{
		AccessModes: accessModes,
		Class:       a.Class,
	})

	if err != nil {
		return err
	}

	fmt.Println(vol.Name)
	return nil
}