package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/spf13/cobra"
)

func NewProjectRm(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ProjectRm{client: c.ClientFactory}, cobra.Command{
		Use: "rm [flags] PROJECT_NAME [PROJECT_NAME...]",
		Example: `
acorn project rm my-project
`,
		SilenceUsage:      true,
		Short:             "Deletes projects",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion(c.ClientFactory)).complete,
	})
	return cmd
}

type ProjectRm struct {
	client ClientFactory
}

func (a *ProjectRm) Run(cmd *cobra.Command, args []string) error {
	for _, projectName := range args {
		if proj, err := project.Remove(cmd.Context(), a.client.Options(), projectName); err != nil {
			return err
		} else if proj != nil {
			fmt.Println(projectName)
		}
	}
	return nil
}
