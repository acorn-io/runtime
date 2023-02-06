package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/spf13/cobra"
)

func NewLogs(c CommandContext) *cobra.Command {
	logs := &Logs{client: c.ClientFactory}
	return cli.Command(logs, cobra.Command{
		Use:               "logs [flags] [APP_NAME|CONTAINER_NAME]",
		SilenceUsage:      true,
		Short:             "Log all workloads from an app",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, appsThenContainersCompletion).withShouldCompleteOptions(onlyNumArgs(1)).complete,
	})

}

type Logs struct {
	Follow bool   `short:"f" usage:"Follow log output"`
	Since  string `short:"s" usage:"Show logs since timestamp (e.g. 42m for 42 minutes)"`
	Tail   int64  `short:"n" usage:"Number of lines in log output"`
	client ClientFactory
}

func (s *Logs) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		app, _, err := appAndArgs(cmd.Context(), c, nil)
		if err != nil {
			return err
		}
		args = []string{app}
	}

	var tailLines *int64
	if s.Tail < 0 {
		err := fmt.Errorf("Tail: Invalid value: %d: must be greater than or equal to 0", s.Tail)
		return err
	} else if s.Tail == 0 {
		tailLines = nil
	} else {
		tailLines = &s.Tail
	}
	return log.Output(cmd.Context(), c, args[0], &client.LogOptions{
		Follow: s.Follow,
		Tail:   tailLines,
		Since:  s.Since,
	})
}
