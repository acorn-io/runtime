package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/acorn-io/aml"
	"github.com/acorn-io/aml/cli/pkg/cmds"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
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
		file = "Acornfile"
	} else if args[0] == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
		result, err := aml.Format(data)
		if err != nil {
			return err
		}
		fmt.Print(string(result))
		return nil
	} else {
		file = args[0]
		if s, err := os.Stat(file); err == nil && s.IsDir() {
			file = filepath.Join(file, "Acornfile")
		}
	}

	fmt := cmds.Fmt{}
	return fmt.Run(cmd, []string{file})
}
