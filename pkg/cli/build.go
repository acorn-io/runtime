package cli

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/merr"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
)

func NewBuild(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Build{client: c.ClientFactory}, cobra.Command{
		Use: "build [flags] DIRECTORY",
		Example: `
# Build from Acornfile file in the local directory
acorn build .`,
		SilenceUsage: true,
		Short:        "Build an app from a Acornfile file",
		Long:         "Build all dependent container and app images from your Acornfile file",
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Build struct {
	ArgsFile string   `usage:"Default args to apply to the build" default:".build-args.acorn"`
	Push     bool     `usage:"Push image after build"`
	File     string   `short:"f" usage:"Name of the build file (default \"DIRECTORY/Acornfile\")"`
	Tag      []string `short:"t" usage:"Apply a tag to the final build"`
	Platform []string `short:"p" usage:"Target platforms (form os/arch[/variant][:osversion] example linux/amd64)"`
	client   ClientFactory
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	if s.Push && (len(s.Tag) == 0 || s.Tag[0] == "") {
		return fmt.Errorf("--push must be used with --tag")
	}

	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	// Check if we can parse the name/tag
	for _, tag := range s.Tag {
		if strings.HasPrefix(tag, "-") {
			return fmt.Errorf("invalid image tag: %v", tag)
		}

		if _, err := name.ParseReference(tag); err != nil {
			return err
		}
	}

	helper := imagesource.NewImageSource(s.client.AcornConfigFile(), s.File, s.ArgsFile, args, s.Platform, false)

	image, _, _, err := helper.GetImageAndDeployArgs(cmd.Context(), c)
	if err != nil {
		return err
	}

	var errs []error
	for _, tag := range s.Tag {
		if err = c.ImageTag(cmd.Context(), image, tag); err != nil {
			errs = append(errs, err)
		}
	}

	if err := merr.NewErrors(errs...); err != nil {
		return err
	}

	fmt.Println(image)

	if s.Push {
		for _, tag := range s.Tag {
			auth, err := getAuthForImage(s.client, tag)
			if err != nil {
				return err
			}
			prog, err := c.ImagePush(cmd.Context(), tag, &client.ImagePushOptions{
				Auth: auth,
			})
			if err != nil {
				return err
			}
			if err := progressbar.Print(prog); err != nil {
				return err
			}
		}
	}

	return nil
}
