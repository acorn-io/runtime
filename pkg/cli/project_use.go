package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/spf13/cobra"
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

	// They want to clear the default
	if args[0] == "" {
		cfg.CurrentProject = ""
		return cfg.Save()
	}

	err = project.Exists(cmd.Context(), a.client.Options().WithCLIConfig(cfg), args[0])
	if err != nil {
		return fmt.Errorf("failed to find project %s, use \"acorn projects\" to list valid project names: %w", args[0], err)
	}

	cfg.CurrentProject = args[0]
	return cfg.Save()
}
