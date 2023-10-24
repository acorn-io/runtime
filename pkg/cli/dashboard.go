package cli

import (
	"fmt"
	"strings"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func NewDashboard(c CommandContext) *cobra.Command {
	return cli.Command(&Dashboard{client: c.ClientFactory}, cobra.Command{
		Use:          "dashboard [flags] [ACORN]",
		SilenceUsage: true,
		Short:        "Open the web dashboard for the project",
		Args:         cobra.MaximumNArgs(1),
	})
}

type Dashboard struct {
	client ClientFactory
}

func (s *Dashboard) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}
	projectName := c.GetProject()
	server, _, ok := strings.Cut(projectName, "/")
	if !ok || !strings.Contains(server, ".") {
		return fmt.Errorf("project [%s] does not have an available dashboard", projectName)
	}
	url := "https://" + projectName

	if len(args) == 1 {
		app, err := c.AppGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		url += "/acorns/" + app.Name
	}

	_ = browser.OpenURL(url)
	fmt.Printf("Opening browser to %s\n", url)
	return err
}
