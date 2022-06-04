package cli

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/pkg/flagparams"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRender() *cobra.Command {
	cmd := cli.Command(&Render{}, cobra.Command{
		Use:          "render [flags] DIRECTORY",
		SilenceUsage: true,
		Short:        "Evaluate and display an acorn.cue with deploy params",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Render struct {
	File string `short:"f" usage:"Name of the dev file" default:"DIRECTORY/acorn.cue"`
}

func (s *Render) Run(cmd *cobra.Command, args []string) error {
	cwd := "."
	if len(args) > 0 {
		cwd = args[0]
	}

	buildFile := build.ResolveFile(s.File, cwd)
	data, err := cue.ReadCUE(buildFile)
	if err != nil {
		return err
	}

	appDef, err := appdefinition.NewAppDefinition(data)
	if err != nil {
		return err
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		return err
	}

	params, err := appDef.DeployParams()
	if err != nil {
		return err
	}

	flags := flagparams.New(s.File, params)
	flags.Usage = usage(appSpec)

	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	appDef, err = appDef.WithDeployParams(deployParams)
	if err != nil {
		return err
	}

	v, err := appDef.JSON()
	if err != nil {
		return err
	}
	fmt.Print(v)

	return nil
}
