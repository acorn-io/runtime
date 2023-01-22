package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewProjectUse(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ProjectUse{client: c.ClientFactory}, cobra.Command{
		Use: "use [flags] PROJECT_NAME",
		Example: `
acorn project use acorn.io/my-user/acorn`,
		SilenceUsage:      true,
		Short:             "Set current project",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion).complete,
	})
	return cmd
}

type ProjectUse struct {
	client ClientFactory
}

func (a *ProjectUse) Run(cmd *cobra.Command, args []string) error {
	cfg, err := config.ReadCLIConfig()
	if err != nil {
		return err
	}

	// They want to clear the default, ok....
	if args[0] == "" {
		cfg.CurrentProject = ""
		return cfg.Save()
	}

	if cfg.ProjectAliases[args[0]] == "" {
		projects, err := project.List(cmd.Context(), a.client.Options().WithCLIConfig(cfg))
		if err != nil {
			return err
		}

		if !slices.Contains(projects, args[0]) {
			return fmt.Errorf("failed to find project %s, use \"acorn projects\" to list valid project names", args[0])
		}
	}

	cfg.CurrentProject = args[0]
	return cfg.Save()
}
