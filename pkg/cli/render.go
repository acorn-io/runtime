package cli

import (
	"fmt"

	"github.com/acorn-io/aml/pkg/cue"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	client2 "github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagesource"
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
	var c client2.Client

	imageAndArgs := imagesource.NewImageSource(s.client.AcornConfigFile(), s.File, args, s.Profile, nil, false)

	_, file, err := imageAndArgs.ResolveImageAndFile()
	if err != nil {
		return err
	}
	if file == "" {
		// Lazily create client so that local file render doesn't require an API connection
		c, err = s.client.CreateDefault()
		if err != nil {
			return err
		}
	}

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
