package cli

import (
	minkserver "github.com/acorn-io/mink/pkg/server"
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	_ "github.com/acorn-io/runtime/pkg/logserver/init"
	"github.com/acorn-io/runtime/pkg/server"
	"github.com/spf13/cobra"
)

var (
	opts = minkserver.DefaultOpts()
)

func NewApiServer(c CommandContext) *cobra.Command {
	api := &APIServer{client: c.ClientFactory}
	cmd := cli.Command(api, cobra.Command{
		Use:          "api-server [flags] [APP_NAME...]",
		SilenceUsage: true,
		Short:        "Run api-server",
		Hidden:       true,
	})
	opts.AddFlags(cmd.Flags())
	return cmd
}

type APIServer struct {
	client ClientFactory
}

func (a *APIServer) Run(cmd *cobra.Command, args []string) error {
	cfg, err := server.New(server.Config{
		Version:     cmd.Version,
		DefaultOpts: opts,
	})
	if err != nil {
		return err
	}

	if err := cfg.Run(cmd.Context()); err != nil {
		return err
	}

	<-cmd.Context().Done()
	return cmd.Context().Err()
}
