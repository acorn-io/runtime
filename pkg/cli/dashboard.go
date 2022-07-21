package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/ui"
	"github.com/spf13/cobra"
)

func NewDashboard() *cobra.Command {
	cmd := cli.Command(&Dashboard{}, cobra.Command{
		Use:          "dashboard",
		SilenceUsage: true,
		Short:        "Dashboard about acorn installation",
		Args:         cobra.NoArgs,
		Hidden:       true,
	})
	return cmd
}

type Dashboard struct {
	ListenAddress string `usage:"Address to locally listen on" default:"127.0.0.1:0"`
	Browser       *bool  `usage:"Open browser after local server starts (default true)"`
}

func (s *Dashboard) Run(cmd *cobra.Command, args []string) error {
	opts := &ui.Options{
		Address: s.ListenAddress,
	}
	if s.Browser == nil {
		opts.OpenBrowser = true
	} else {
		opts.OpenBrowser = *s.Browser
	}

	return ui.UI(cmd.Context(), opts)
}
