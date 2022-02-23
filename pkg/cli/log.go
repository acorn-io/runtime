package cli

import (
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/log"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/rancher/wrangler-cli"
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
	Follow bool `short:"f" desc:"Follow log output"`
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

	return log.App(cmd.Context(), &app, &log.Options{
		Client: c,
	})
}
