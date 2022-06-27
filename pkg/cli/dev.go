package cli

import (
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/dev"
	"github.com/spf13/cobra"
)

func NewDev() *cobra.Command {
	cmd := cli.Command(&Dev{}, cobra.Command{
		Use:          "dev [flags] DIRECTORY",
		SilenceUsage: true,
		Short:        "Build and run an app in development mode",
		Long:         "Build and run an app in development mode",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.AddCommand(NewRender())
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Dev struct {
	File string `short:"f" usage:"Name of the dev file" default:"DIRECTORY/acorn.cue"`
	RunArgs
}

func (s *Dev) Run(cmd *cobra.Command, args []string) error {
	cwd := "."
	if len(args) > 0 {
		cwd = args[0]
	}

	c, err := client.Default()
	if err != nil {
		return err
	}

	opts, err := s.ToOpts()
	if err != nil {
		return err
	}

	return dev.Dev(cmd.Context(), s.File, &dev.Options{
		Args:   args,
		Client: c,
		Build: build.Options{
			Cwd:      cwd,
			Profiles: opts.Profiles,
		},
		Run:       opts,
		Dangerous: s.Dangerous,
	})
}
