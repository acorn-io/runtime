package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/spf13/cobra"
)

func NewProjectUpdate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&ProjectUpdate{client: c.ClientFactory}, cobra.Command{
		Use: "update [flags] PROJECT_NAME",
		Example: `
acorn project update my-project
`,
		SilenceUsage:      true,
		Short:             "Update project",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, projectsCompletion).complete,
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

type ProjectUpdate struct {
	client           ClientFactory
	DefaultRegion    string   `usage:"Default region for project resources"`
	SupportedRegions []string `name:"supported-region" usage:"Supported regions for the created project"`
}

func (a *ProjectUpdate) Run(cmd *cobra.Command, args []string) error {
	projectsDetails, err := project.GetDetails(cmd.Context(), project.Options{}, []string{args[0]})
	if err != nil {
		return err
	}
	if err := project.Update(cmd.Context(), a.client.Options(), projectsDetails[0], a.DefaultRegion, a.SupportedRegions); err != nil {
		return err
	} else {
		fmt.Println(projectsDetails[0].FullName)
	}
	return nil
}
