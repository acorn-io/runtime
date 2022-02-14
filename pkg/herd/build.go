package herd

import (
	"fmt"
	"os"

	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewBuild() *cobra.Command {
	return cli.Command(&Build{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Build an app from a herd.cue file",
		Long:         "Build all dependent container and app images from you herd.cue file",
		Args:         cobra.RangeArgs(0, 1),
	})
}

type Build struct {
	File string `short:"f" desc:"Name of the build file" default:"herd.cue"`
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	var (
		cwd string
		err error
	)

	if len(args) == 0 {
		cwd, err = os.Getwd()
		if err != nil {
			return err
		}
	} else {
		cwd = args[0]
	}

	image, err := build.Build(cmd.Context(), s.File, &build.Opts{
		Cwd: cwd,
	})
	if err != nil {
		return err
	}

	fmt.Println(image.ID)
	return err
}
