package cli

import (
	"fmt"
	"strings"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/spf13/cobra"
)

func NewProject(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Project{client: c.ClientFactory}, cobra.Command{
		Use:     "project [flags]",
		Aliases: []string{"projects", "["},
		Example: `
acorn project`,
		SilenceUsage: true,
		Short:        "Manage projects",
		Args:         cobra.MaximumNArgs(1),
	})
	cmd.AddCommand(NewProjectUse(c))
	return cmd
}

type Project struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

type projectEntry struct {
	Name        string `json:"name,omitempty"`
	Default     bool   `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

func (a *Project) Run(cmd *cobra.Command, args []string) error {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	projects, err := project.List(cmd.Context(), cfg, a.client.Options())
	if err != nil {
		return err
	}

	defaultProject := cfg.CurrentProject

	c, err := project.Client(cmd.Context(), a.client.Options())
	if err == nil {
		defaultProject = c.GetProject()
	}

	out := table.NewWriter(tables.ProjectClient, a.Quiet, a.Output)
	for _, project := range projects {
		if len(args) == 1 && !strings.HasPrefix(project, args[0]+"/") {
			continue
		}
		out.Write(projectEntry{
			Name:    project,
			Default: defaultProject == project,
		})
	}

	for _, entry := range typed.Sorted(cfg.ProjectAliases) {
		out.Write(projectEntry{
			Name:        entry.Key,
			Default:     defaultProject == entry.Value,
			Description: fmt.Sprintf("alias to %s", entry.Value),
		})
	}

	return out.Err()
}
