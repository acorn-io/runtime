package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewRegions(c CommandContext) *cobra.Command {
	return cli.Command(&Regions{client: c.ClientFactory}, cobra.Command{
		Use:     "regions [flags] [REGION...]",
		Aliases: []string{"region"},
		Example: `
acorn offering regions`,
		SilenceUsage:      true,
		Short:             "List available regions",
		ValidArgsFunction: newCompletion(c.ClientFactory, regionsCompletion).withoutProjectCompletion().complete,
	})
}

type Regions struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (a *Regions) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Region, a.Quiet, a.Output)

	if len(args) == 1 {
		region, err := c.RegionGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(region)
		return out.Err()
	}

	regions, err := c.RegionList(cmd.Context())
	if err != nil {
		return err
	}

	for _, region := range regions {
		if len(args) > 0 {
			if slices.Contains(args, region.Name) {
				out.Write(region)
			}
		} else {
			out.Write(&region)
		}
	}

	return out.Err()
}
