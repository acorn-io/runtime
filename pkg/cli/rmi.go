package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewRmi() *cobra.Command {
	return cli.Command(&Rmi{}, cobra.Command{
		Use: "rmi [flags] [IMAGE_NAME|TAG...]",
		Example: `
acorn rmi some-image`,
		SilenceUsage: true,
		Short:        "Delete an image or tag",
	})
}

type Rmi struct {
	Volumes bool `usage:"Delete volumes" short:"v"`
}

func (a *Rmi) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, arg := range args {
		image, err := client.ImageDelete(cmd.Context(), arg)
		if err != nil {
			return fmt.Errorf("deleting volume %s: %w", arg, err)
		}
		if image != nil {
			fmt.Println(arg)
		}
	}

	return nil
}
