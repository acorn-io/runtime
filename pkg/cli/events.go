package cli

import (
	// "fmt"
	"time"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	// "github.com/acorn-io/acorn/pkg/client"
	// "github.com/acorn-io/acorn/pkg/imagesource"
	// "github.com/acorn-io/acorn/pkg/progressbar"
	// "github.com/google/go-containerregistry/pkg/name"
	// "github.com/rancher/wrangler/pkg/merr"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewEvent(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Events{client: c.ClientFactory}, cobra.Command{
		Use:          "events [flags] FILTER",
		SilenceUsage: true,
		Short:        "List events about Acorn resources",
		Args:         cobra.MaximumNArgs(1),
		// ValidArgsFunction: newCompletion(c.ClientFactory, imagesCompletion(true)).withSuccessDirective(cobra.ShellCompDirectiveDefault).withShouldCompleteOptions(onlyNumArgs(1)).complete,
		Example: `# List all events in the current project
  acorn events

  # List events across all projects
  acorn -A events

  # Stream events in the current project
  acorn events -w

  # List the last 10 events 
  acorn events --tail 10

# Filtering by Resource 
  # The FILTER argument comes in the form '<kind>/<name>' and, when given, prunes the result set to contain events related to Acorn resources with the respective kind and name.
  # List events related to the 'hello' app in the current project
  acorn events app/hello
  
  # The '/<name>' suffix is optional and, when omitted, matches events related to any resource of the given kind.
  # List events related to any volume in the current project
  acorn events vol 

# Filtering by Time
  # The --since and --until options can be Unix timestamps, date formatted timestamps, or Go duration strings (relative to system time).
  # List events observed within the last 15 minutes 
  acorn events --since 15m

  # List events observed between 2023-05-08T15:04:05 and 2023-05-08T15:05:05 (inclusive)
  acorn events --since '2023-05-08T15:04:05' --until '2023-05-08T15:05:05'

# Getting Event Context 
  # The 'context' field provides additional information about an event.
  # By default, this field is elided from this command's output, but can be enabled via the '--with-context' field.
  acorn events --with-context
`})
	return cmd
}

type Events struct {
	// TODO(njhale): Fix flags/options
	Since       *time.Duration `usage:"Show all events created since timestamp" short:"s"`
	Until       *time.Duration `usage:"Stream events until this timestamp" short:"u"`
	WithContext bool           `usage:"Don't strip the event context from response" short:"c"`
	Watch       bool           `usage:"Stream events" short:"w"`
	Tail        *int           `usage:"Return this number of latest events" short:"t"`
	Output      string         `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client      ClientFactory
}

func (e *Events) Run(cmd *cobra.Command, args []string) error {
	c, err := e.client.CreateDefault()
	if err != nil {
		return err
	}

	// TODO(njhale): Implement filters, watch, etc
	out := table.NewWriter(tables.Event, false, e.Output)
	events, err := c.EventList(cmd.Context())
	if err != nil {
		out.Write(err)
	} else {
		for _, event := range events {
			out.Write(event)
		}
	}

	return out.Err()
}
