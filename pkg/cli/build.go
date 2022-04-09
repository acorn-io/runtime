package cli

import (
	"fmt"

	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewBuild() *cobra.Command {
	return cli.Command(&Build{}, cobra.Command{
		Use: "build [flags] DIRECTORY",
		Example: `
# Build from herd.cue file in the local directory
herd build .`,
		SilenceUsage: true,
		Short:        "Build an app from a herd.cue file",
		Long:         "Build all dependent container and app images from your herd.cue file",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Build struct {
	File string `short:"f" desc:"Name of the build file" default:"DIRECTORY/herd.cue"`
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	cwd := args[0]

	image, err := build.Build(cmd.Context(), s.File, &build.Options{
		Cwd: cwd,
	})
	if err != nil {
		return err
	}

	fmt.Println(image.ID)
	return err
}
