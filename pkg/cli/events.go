package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/spf13/cobra"
)

func NewEvent(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Events{client: c.ClientFactory}, cobra.Command{
		Use:               "events [flags] [PREFIX]",
		SilenceUsage:      true,
		Short:             "List events about Acorn resources",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, eventsCompletion).complete,
		Example: `# List all events in the current project
  acorn events

  # List events across all projects
  acorn -A events


  # List the last 10 events 
  acorn events --tail 10

  # List the last 5 events and follow the event log
  acorn events --tail 5 -f

  # Filter by Event Source 
  # If a PREFIX is given in the form '<kind>/<name>', the results of this command are pruned to include
  # only those events sourced by resources matching the given kind and name.
  # List events sourced by the 'hello' app in the current project
  acorn events app/hello
  
  # If the '/<name>' suffix is omitted, '<kind>' will match events sourced by any resource of the given kind.
  # List events related to any app in the current project
  acorn events app 

  # Filter by Event Name
  # If the PREFIX '/<name>' suffix is omitted, and the '<kind>' doesn't match a known event source, its value
  # is interpreted as an event name prefix.
  # List events with names that begin with '4b2b' 
  acorn events 4b2b

  # Get a single event by name
  acorn events 4b2ba097badf2031c4718609b9179fb5
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
		Tail:   e.Tail,
		Follow: e.Follow,
	}

	if len(args) > 0 {
		opts.Prefix = args[0]
	}

	events, err := c.EventStream(cmd.Context(), opts)
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Event, false, e.Output)
	for event := range events {
		out.Write(&event)

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
