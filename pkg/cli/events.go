package cli

import (
	// "fmt"

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
		Long: "List events about acorn resources",
		// Args: cobra.MinimumNArgs(1),
	})
	// cmd.Flags().SetInterspersed(false)
	return cmd
}

type Events struct {
	// TODO(njhale): Add flags/options
	// Push     bool     `usage:"Push image after build"`
	// File     string   `short:"f" usage:"Name of the build file (default \"DIRECTORY/Acornfile\")"`
	// Tag      []string `short:"t" usage:"Apply a tag to the final build"`
	// Platform []string `short:"p" usage:"Target platforms (form os/arch[/variant][:osversion] example linux/amd64)"`
	// Profile  []string `usage:"Profile to assign default values"`
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
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
