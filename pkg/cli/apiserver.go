package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/server"
	"github.com/spf13/cobra"
)

var (
	apiServer = server.New()
)

func NewApiServer(c CommandContext) *cobra.Command {
	api := &APIServer{client: c.ClientFactory}
	cmd := cli.Command(api, cobra.Command{
		Use:          "api-server [flags] [APP_NAME...]",
		SilenceUsage: true,
		Short:        "Run api-server",
		Hidden:       true,
	})
	apiServer.AddFlags(cmd.Flags())
	return cmd
}

type APIServer struct {
	client ClientFactory
}

func (a *APIServer) Run(cmd *cobra.Command, args []string) error {
	cfg, err := apiServer.NewConfig(cmd.Version)
	if err != nil {
		return err
	}

	return apiServer.Run(cmd.Context(), cfg)
}
