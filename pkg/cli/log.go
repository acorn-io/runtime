package cli

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/acorn/pkg/system"
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
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
	Follow bool `short:"f" usage:"Follow log output"`
}

func (s *Logs) Run(cmd *cobra.Command, args []string) error {
	c, err := hclient.Default()
	if err != nil {
		return err
	}

	var (
		ns  = system.UserNamespace()
		app v1.AppInstance
	)

	if err := c.Get(cmd.Context(), hclient.ObjectKey{Namespace: ns, Name: args[0]}, &app); err != nil {
		return err
	}

	return log.Output(cmd.Context(), &app, &log.Options{
		Client:   c,
		NoFollow: !s.Follow,
	})
}
