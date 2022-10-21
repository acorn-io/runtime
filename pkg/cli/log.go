package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/spf13/cobra"
)

func NewLogs() *cobra.Command {
	return cli.Command(&Logs{}, cobra.Command{
		Use:          "logs [flags] APP_NAME",
		SilenceUsage: true,
		Short:        "Log all pods from app",
		Args:         cobra.RangeArgs(1, 1),
	})
}

type Logs struct {
	Follow    bool  `short:"f" usage:"Follow log output"`
	TailLines int64 `short:"n" usage:"Number of lines in log output"`
}

func (s *Logs) Run(cmd *cobra.Command, args []string) error {
	c, err := client.Default()
	if err != nil {
		return err
	}
	var tailLines *int64
	if s.TailLines == 0 {
		tailLines = nil
	} else {
		tailLines = &s.TailLines
	}

	return log.Output(cmd.Context(), c, args[0], &client.LogOptions{
		Follow:    s.Follow,
		TailLines: tailLines,
	})
}
