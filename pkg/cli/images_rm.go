package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
}

func (a *ImageDelete) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, image := range args {
		img, removedTags, err := c.ImageDelete(cmd.Context(), image, &client.ImageDeleteOptions{Force: a.Force})
		if err != nil {
			if apierrors.IsNotFound(err) {
				fmt.Printf("Error: No such image: %s\n", image)
				continue
			}
			return fmt.Errorf("deleting %s: %w", image, err)
		}

		if img == nil && len(removedTags) == 0 {
			logrus.Infof("No image found for %s", image)
			continue // no idea how this could happen anyway
		}
		for _, tag := range removedTags {
			fmt.Printf("Untagged %s\n", tag)
		}
		if img != nil {
			fmt.Printf("Deleted %s\n", img.Name)
		}
	}

	return nil
}
