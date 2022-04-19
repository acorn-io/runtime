package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/rancher/wrangler-cli"
	"github.com/rancher/wrangler/pkg/merr"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewBuild() *cobra.Command {
	cmd := cli.Command(&Build{}, cobra.Command{
		Use: "build [flags] DIRECTORY",
		Example: `
# Build from herd.cue file in the local directory
herd build .`,
		SilenceUsage: true,
		Short:        "Build an app from a herd.cue file",
		Long:         "Build all dependent container and app images from your herd.cue file",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Build struct {
	File string   `short:"f" desc:"Name of the build file" default:"DIRECTORY/herd.cue"`
	Tag  []string `short:"t" desc:"Apply a tag to the final build"`
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

	image, err := build.Build(cmd.Context(), s.File, &build.Options{
		Cwd:    cwd,
		Params: params,
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
