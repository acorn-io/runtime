package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewComputeClasses(c CommandContext) *cobra.Command {
	return cli.Command(&ComputeClass{client: c.ClientFactory}, cobra.Command{
		Use:     "computeclasses [flags] [COMPUTECLASS_NAME...]",
		Aliases: []string{"computeclass", "wc", "workload"},
		Example: `
acorn computeclasses`,
		SilenceUsage:      true,
		Short:             "List available ComputeClasses",
		ValidArgsFunction: newCompletion(c.ClientFactory, computeClassCompletion).complete,
	})
}

type ComputeClass struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (a *ComputeClass) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.ComputeClass, a.Quiet, a.Output)

	if len(args) == 1 {
		cc, err := c.ComputeClassGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(cc)
		return out.Err()
	}

	computeClasses, err := c.ComputeClassList(cmd.Context())
	if err != nil {
		return err
	}

	for _, cc := range computeClasses {
		if len(args) == 0 || slices.Contains(args, cc.Name) {
			out.Write(&cc)
		}
	}

	return out.Err()
}
