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
		ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).checkProjectPrefix().complete,
	})
	return cmd
}

type ImageDelete struct {
	client ClientFactory
	Force  bool `usage:"Force Delete" short:"f"`
}

func (a *ImageDelete) Run(cmd *cobra.Command, args []string) error {
	for _, image := range args {
		// if the image is of the form project::image,
		// parse off project:: and change the client
		c, image, err := parseArgGetClient(a.client, cmd, image)
		if err != nil {
			return err
		}

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
