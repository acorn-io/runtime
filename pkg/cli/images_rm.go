package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewImageDelete() *cobra.Command {
	cmd := cli.Command(&ImageDelete{}, cobra.Command{
		Use: "rm [IMAGE_NAME...]",
		Example: `
acorn image rm my-image`,
		SilenceUsage: true,
		Short:        "Delete an Image",
	})
	return cmd
}

type ImageDelete struct {
}

func (a *ImageDelete) Run(cmd *cobra.Command, args []string) error {
	client, err := client.Default()
	if err != nil {
		return err
	}

	for _, image := range args {
		deleted, err := client.ImageDelete(cmd.Context(), image)
		if err != nil {
			return fmt.Errorf("deleting %s: %w", image, err)
		}
		if deleted != nil {
			fmt.Println(image)
		} else {
			fmt.Printf("Error: No such image: %s\n", image)
		}
	}

	return nil
}
