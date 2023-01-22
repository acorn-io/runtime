package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/spf13/cobra"
)

func NewImageDelete(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ImageDelete{client: c.ClientFactory}, cobra.Command{
		Use:               "rm [IMAGE_NAME...]",
		Example:           `acorn image rm my-image`,
		SilenceUsage:      true,
		Short:             "Delete an Image",
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).complete,
	})
	return cmd
}

type ImageDelete struct {
	client ClientFactory
	Force  bool `usage:"Force Delete" short:"f"`
	All    bool `usage:"Delete all images" short:"a"`
}

func (a *ImageDelete) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	var imageNames []string

	if a.All {
		images, err := c.ImageList(cmd.Context())
		if err != nil {
			return err
		}

		for _, image := range images {
			imageNames = append(imageNames, image.Name)
		}
	} else {
		imageNames = args
	}

	for _, image := range imageNames {
		deleted, err := c.ImageDelete(cmd.Context(), image, &client.ImageDeleteOptions{Force: a.Force})

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
