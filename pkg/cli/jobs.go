package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/spf13/cobra"
)

func NewJob(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Job{client: c.ClientFactory}, cobra.Command{
		Use:     "job [flags] [ACORN_NAME|JOB_NAME...]",
		Aliases: []string{"jobs"},
		Example: `
acorn jobs`,
		SilenceUsage:      true,
		Short:             "Manage jobs",
		ValidArgsFunction: newCompletion(c.ClientFactory, jobsCompletion).complete,
	})
	cmd.AddCommand(NewJobRestart(c))
	return cmd
}

type Job struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (a *Job) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Job, a.Quiet, a.Output)

	jobs, err := c.JobList(cmd.Context(), nil)
	if err != nil {
		return err
	}

	// Build a map of args to use instead of a slice for faster lookups.
	argsMap := map[string]bool{}
	for _, arg := range args {
		argsMap[arg] = true
	}

	printed := map[string]bool{}
	for _, job := range jobs {
		appName, _ := publicname.Split(job.Name)

		// If args were passed and this job doesn't match any args or has already been printed, skip it.
		matchesArg := argsMap[appName] || argsMap[job.Name]
		if len(args) != 0 && (!matchesArg || printed[job.Name]) {
			continue
		}

		printed[job.Name] = true
		out.Write(&job)
	}

	return out.Err()
}
