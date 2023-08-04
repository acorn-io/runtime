package cli

import (
	"fmt"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewJobRestart(c CommandContext) *cobra.Command {
	cd := &JobRestart{client: c.ClientFactory}
	cmd := cli.Command(cd, cobra.Command{
		Use: "restart [JOB_NAME...]",
		Example: `
acorn job restart app-name.job-name`,
		SilenceUsage:      true,
		Short:             "Restart a job",
		Aliases:           []string{"rs"},
		ValidArgsFunction: newCompletion(c.ClientFactory, jobsCompletion).complete,
	})
	return cmd
}

type JobRestart struct {
	client ClientFactory
}

func (a *JobRestart) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	for _, job := range args {
		err := c.JobRestart(cmd.Context(), job)
		if err != nil {
			return fmt.Errorf("restarting %s: %w", job, err)
		}
	}

	return nil
}
