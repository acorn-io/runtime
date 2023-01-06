package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewRender(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Render{client: c.ClientFactory}, cobra.Command{
		Use:          "render [flags] DIRECTORY [acorn args]",
		SilenceUsage: true,
		Short:        "Evaluate and display an Acornfile with args",
		Args:         cobra.MinimumNArgs(1),
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Render struct {
	File    string   `short:"f" usage:"Name of the dev file" default:"DIRECTORY/Acornfile"`
	Profile []string `usage:"Profile to assign default values"`
	Output  string   `usage:"Output in JSON or YAML" default:"json" short:"o"`
	client  ClientFactory
}

func (s *Render) Run(cmd *cobra.Command, args []string) error {
	cwd := "."
	if len(args) > 0 {
		cwd = args[0]
	}

	appDef, flags, err := deployargs.ToFlagsFromFile(s.File, cwd)
	if err != nil {
		return err
	}

	deployParams, err := flags.Parse(args)
	if pflag.ErrHelp == err {
		return nil
	} else if err != nil {
		return err
	}

	appDef, _, err = appDef.WithArgs(deployParams, s.Profile)
	if err != nil {
		return err
	}

	var v string
	switch s.Output {
	case "yaml":
		v, err = appDef.YAML()
	case "json":
		if v, err = appDef.JSON(); err == nil {
			v += "\n" // appDef.YAML() appends a line break
		}
	default:
		return fmt.Errorf("unsupported output format %s", s.Output)
	}

	if err != nil {
		return err
	}
	fmt.Print(v)
	return nil
}
