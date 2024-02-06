package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/spf13/cobra"
)

func NewProjectCreate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ProjectCreate{client: c.ClientFactory}, cobra.Command{
		Use: "create [flags] PROJECT_NAME [PROJECT_NAME...]",
		Example: `
# Create a project locally
acorn project create my-new-project

# Create a project on remote service acorn.io
acorn project create acorn.io/username/new-project
`,
		SilenceUsage:      true,
		Short:             "Create new project",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion(c.ClientFactory)).complete,
	})
	// This will produce an error if the region flag doesn't exist or a completion function has already
	// been registered for this flag. Not returning the error since neither of these is likely occur.
	if err := cmd.RegisterFlagCompletionFunc("default-region", newCompletion(c.ClientFactory, regionsCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for --default-region flag: %v\n", err)
	}
	if err := cmd.RegisterFlagCompletionFunc("supported-region", newCompletion(c.ClientFactory, regionsCompletion).complete); err != nil {
		cmd.Printf("Error registering completion function for --supported-region flag: %v\n", err)
	}
	return cmd
}

type ProjectCreate struct {
	client           ClientFactory
	DefaultRegion    string   `usage:"Default region for project resources"`
	SupportedRegions []string `name:"supported-region" usage:"Supported regions for created project"`
}

func (a *ProjectCreate) Run(cmd *cobra.Command, args []string) error {
	for _, projectName := range args {
		if err := project.Create(cmd.Context(), a.client.Options(), projectName, a.DefaultRegion, a.SupportedRegions); err != nil {
			return err
		}
		fmt.Println(projectName)
	}
	return nil
}
