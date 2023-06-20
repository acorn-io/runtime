package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/progressbar"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/rancher/wrangler/pkg/merr"
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
	Push     bool     `usage:"Push image after build"`
	File     string   `short:"f" usage:"Name of the build file (default \"DIRECTORY/Acornfile\")"`
	Tag      []string `short:"t" usage:"Apply a tag to the final build"`
	Platform []string `short:"p" usage:"Target platforms (form os/arch[/variant][:osversion] example linux/amd64)"`
	Profile  []string `usage:"Profile to assign default values"`
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

	helper := imagesource.NewImageSource(s.File, args, s.Profile, s.Platform, false)
	image, _, err := helper.GetImageAndDeployArgs(cmd.Context(), c)
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
			parsedTag, err := name.NewTag(tag)
			if err != nil {
				return err
			}
			creds, err := imagesource.GetCreds(c)
			if err != nil {
				return err
			}
			auth, _, err := creds(cmd.Context(), parsedTag.RegistryStr())
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
