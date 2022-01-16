package herd

import (
	"fmt"

	"github.com/rancher/wrangler-cli"
	"github.com/spf13/cobra"
)

func NewBuild() *cobra.Command {
	return cli.Command(&Build{}, cobra.Command{
		Short: "Build an app from a herd.cue file",
		Long:  "Build all dependent container and app images from you herd.cue file",
	})
}

type Build struct {
}

func (s *Build) Run(cmd *cobra.Command, args []string) error {
	fmt.Println("I do stuff")
	return nil
}
