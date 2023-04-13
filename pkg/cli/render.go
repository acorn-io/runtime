package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/imagesource"
	"github.com/acorn-io/aml/pkg/cue"
	"github.com/spf13/cobra"
)

func NewRender(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Render{client: c.ClientFactory}, cobra.Command{
		Use:          "render [flags] DIRECTORY [acorn args]",
		SilenceUsage: true,
		Short:        "Evaluate and display an Acornfile with args",
	})
	cmd.Flags().SetInterspersed(false)
	return cmd
}

type Render struct {
	File    string   `short:"f" usage:"Name of the dev file (default \"DIRECTORY/Acornfile\")"`
	Profile []string `usage:"Profile to assign default values"`
	Output  string   `usage:"Output in JSON or YAML" default:"aml" short:"o"`
	client  ClientFactory
}

func (s *Render) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	imageAndArgs := imagesource.NewImageSource(s.File, args, s.Profile, nil)

	appDef, _, err := imageAndArgs.GetAppDefinition(cmd.Context(), c)
	if err != nil {
		return err
	}

	var v string
	switch s.Output {
	case "yaml":
		v, err = appDef.YAML()
	case "aml":
		v, err = appDef.JSON()
		if err != nil {
			return err
		}
		var d []byte
		d, err = cue.FmtBytes([]byte(v))
		v = string(d)
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
