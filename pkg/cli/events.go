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
		Use:               "events [flags] [EVENT_NAME]",
		SilenceUsage:      true,
		Short:             "List events about Acorn resources",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, eventsCompletion).complete,
		Example: `# List all events in the current project
  acorn events

  # List events across all projects
  acorn -A events

  # Get a single event by name
  acorn events 4b2ba097badf2031c4718609b9179fb5

  # List the last 10 events 
  acorn events --tail 10

  # List the last 5 events and follow the event log
  acorn events --tail 5 -f

  # Getting Details 
  # The 'details' field provides additional information about an event.
  # By default, this field is elided from this command's output, but can be enabled via the '--details' flag.
  # This flag must be used in conjunction with a non-table output format, like '-o=yaml'.
  acorn events --details -o yaml
`})
	return cmd
}

type Events struct {
	Tail    int    `usage:"Return this number of latest events" short:"t"`
	Follow  bool   `usage:"Follow the event log" short:"f"`
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
		Follow:  e.Follow,
		Details: e.Details,
	}

	if len(args) > 0 {
		opts.Name = args[0]
	}

	events, err := c.EventStream(cmd.Context(), opts)
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Event, false, e.Output)
	for event := range events {
		out.Write(event)

		if !opts.Follow {
			// Wait to flush until all events have been written.
			// This ensures consistent column width for table formatting.
			continue
		}

		if err := out.Flush(); err != nil {
			break
		}
	}

	return out.Err()
}
