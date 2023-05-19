package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewEvent(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Events{client: c.ClientFactory}, cobra.Command{
		Use:          "events [flags]",
		SilenceUsage: true,
		Short:        "List events about Acorn resources",
		Args:         cobra.MaximumNArgs(0),
		Example: `# List all events in the current project
  acorn events

  # List events across all projects
  acorn -A events

  # List the last 10 events 
  acorn events --tail 10

# Getting Details 
  # The 'details' field provides additional information about an event.
  # By default, this field is elided from this command's output, but can be enabled via the '--details' flag.
  acorn events --details
`})
	return cmd
}

type Events struct {
	Tail    int    `usage:"Return this number of latest events" short:"t"`
	Details bool   `usage:"Don't strip event details from response" short:"d"`
	Output  string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client  ClientFactory
}

func (e *Events) Run(cmd *cobra.Command, args []string) error {
	c, err := e.client.CreateDefault()
	if err != nil {
		return err
	}

	opts := &client.EventStreamOptions{
		Tail:    e.Tail,
		Details: e.Details,
	}

	events, err := c.EventStream(cmd.Context(), opts)
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Event, false, e.Output)
	for event := range events {
		out.Write(event)
	}

	return out.Err()
}
