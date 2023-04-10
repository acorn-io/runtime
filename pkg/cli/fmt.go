package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/aml/pkg/cue"
	"github.com/spf13/cobra"
)

func NewFmt(_ CommandContext) *cobra.Command {
	return cli.Command(&Fmt{}, cobra.Command{
		Use:          "fmt [flags] [ACORNFILE]",
		SilenceUsage: true,
		Short:        "Format an Acornfile",
		Args:         cobra.MaximumNArgs(1),
	})
}

type Fmt struct {
}

func (s *Fmt) Run(cmd *cobra.Command, args []string) error {
	var (
		file string
	)
	if len(args) == 0 {
		file = build.FindAcornCue(".")
	} else if args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		result, err := cue.FmtBytes(data)
		if err != nil {
			return err
		}
		fmt.Print(string(result))
		return nil
	} else {
		file = args[0]
		if s, err := os.Stat(file); err == nil && s.IsDir() {
			file = build.FindAcornCue(file)
		}
	}

	data, err := cue.ReadCUE(file)
	if err != nil {
		return err
	}

	_, err = appdefinition.NewAppDefinition(data)
	if err != nil {
		return err
	}

	_, err = cue.FmtCUEInPlace(file)
	return err
}
