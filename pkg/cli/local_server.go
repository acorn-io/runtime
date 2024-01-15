package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/spf13/cobra"
)

func NewLocalServer(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Server{}, cobra.Command{
		SilenceUsage: true,
		Short:        "Run local development server",
	})
	return cmd
}

type Server struct {
}

func (a *Server) Run(cmd *cobra.Command, args []string) error {
	return local.ServerRun(cmd.Context())
}
