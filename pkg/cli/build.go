package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewBuild() *cobra.Command {
	cmd := cli.Command(&Build{}, cobra.Command{
		Use: "build [flags] DIRECTORY",
		Example: `
# Build from acorn.cue file in the local directory
acorn build .`,
		SilenceUsage: true,
		Short:        "Build an app from a acorn.cue file",
		Long:         "Build all dependent container and app images from your acorn.cue file",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Build struct {
	File      string   `short:"f" desc:"Name of the build file" default:"DIRECTORY/acorn.cue"`
	Tag       []string `short:"t" desc:"Apply a tag to the final build"`
	Platforms []string `short:"p" desc:"Target platforms (form os/arch[/variant][:osversion] example linux/amd64)"`
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
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

	platforms, err := build.ParsePlatforms(s.Platforms)
	if err != nil {
		return err
	}

	image, err := build.Build(cmd.Context(), s.File, &build.Options{
		Cwd:       cwd,
		Params:    params,
		Platforms: platforms,
	})
	if err != nil {
		return err
	}

	var errs []error
	for _, tag := range s.Tag {
		_, err := c.Tag(cmd.Context(), image.ID, tag)
		if err != nil {
			errs = append(errs, err)
		}
	}

	fmt.Println(image.ID)
	return merr.NewErrors(errs...)
}
