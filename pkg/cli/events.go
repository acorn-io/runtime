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
		Use: "events [flags]",
		// TODO(njhale): Add examples
		Example: `
# List Acorn events.
`,
		SilenceUsage: true,
		Short:        "List Acorn events",
		// TODO(njhale): better long description
		Long: "List events about Acorn resources",
		// Args: cobra.MinimumNArgs(1),
	})
	// cmd.Flags().SetInterspersed(false)
	return cmd
}

type Events struct {
	// TODO(njhale): Fix flags/options
	Filter string         `usage:"Filter output based on conditions provided" short:"f"`
	Since  *time.Duration `usage:"Show all events created since timestamp" short:"s"`
	Until  *time.Duration `usage:"Stream events until this timestamp" short:"u"`
	Quiet  bool           `usage:"Output only names" short:"q"`
	Output string         `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (e *Events) Run(cmd *cobra.Command, args []string) error {
	c, err := e.client.CreateDefault()
	if err != nil {
		return err
	}

	// TODO(njhale): Implement filters, watch, etc
	out := table.NewWriter(tables.Event, e.Quiet, e.Output)
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
