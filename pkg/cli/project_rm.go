package cli

import (
	"bufio"
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
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion(c.ClientFactory)).complete,
	})
	return cmd
}

type ProjectRm struct {
	Stdin  bool `usage:"Take project names from stdin"`
	client ClientFactory
}

func (a *ProjectRm) Run(cmd *cobra.Command, args []string) error {
	projectNames := make([]string, 0)
	if a.Stdin {
		scanner := bufio.NewScanner(cmd.InOrStdin())
		scanner.Split(bufio.ScanWords)
		for scanner.Scan() {
			projectNames = append(projectNames, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	} else {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		projectNames = append(projectNames, args...)
	}
	for _, projectName := range projectNames {
		if proj, err := project.Remove(cmd.Context(), a.client.Options(), projectName); err != nil {
			return err
		} else if proj != nil {
			fmt.Println(projectName)
		}
	}
	return nil
}
