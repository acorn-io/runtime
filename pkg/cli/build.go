package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/progressbar"
	"github.com/acorn-io/acorn/pkg/streams"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewBuild(c client.CommandContext) *cobra.Command {
	cmd := cli.Command(&Build{client: c.ClientFactory}, cobra.Command{
		Use: "build [flags] DIRECTORY",
		Example: `
# Build from Acornfile file in the local directory
acorn build .`,
		SilenceUsage: true,
		Short:        "Build an app from a Acornfile file",
		Long:         "Build all dependent container and app images from your Acornfile file",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Build struct {
	Push     bool     `usage:"Push image after build"`
	File     string   `short:"f" usage:"Name of the build file" default:"DIRECTORY/Acornfile"`
	Tag      []string `short:"t" usage:"Apply a tag to the final build"`
	Platform []string `short:"p" usage:"Target platforms (form os/arch[/variant][:osversion] example linux/amd64)"`
	Profile  []string `usage:"Profile to assign default values"`
	client   client.ClientFactory
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	if s.Push && (len(s.Tag) == 0 || s.Tag[0] == "") {
		return fmt.Errorf("--push must be used with --tag")
	}
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	cwd := args[0]

	params, err := build.ParseParams(s.File, cwd, args)
	if err == pflag.ErrHelp {
		return nil
	} else if err != nil {
		return err
	}

	platforms, err := build.ParsePlatforms(s.Platform)
	if err != nil {
		return err
	}

	image, err := c.AcornImageBuild(cmd.Context(), s.File, &client.AcornImageBuildOptions{
		Cwd:       cwd,
		Args:      params,
		Platforms: platforms,
		Profiles:  s.Profile,
		Streams:   &streams.Current().Output,
	})
	if err != nil {
		return err
	}

	var errs []error
	for _, tag := range s.Tag {
		if err = c.ImageTag(cmd.Context(), image.ID, tag); err != nil {
			errs = append(errs, err)
		}
	}

	if err := merr.NewErrors(errs...); err != nil {
		return err
	}

	fmt.Println(image.ID)

	if s.Push {
		for _, tag := range s.Tag {
			prog, err := c.ImagePush(cmd.Context(), tag, nil)
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
