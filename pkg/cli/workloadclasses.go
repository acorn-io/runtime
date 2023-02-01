package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewWorkloadClasses(c CommandContext) *cobra.Command {
	return cli.Command(&WorkloadClass{client: c.ClientFactory}, cobra.Command{
		Use:     "workloadclasses [flags] [APP_NAME...]",
		Aliases: []string{"workloadclass", "wc", "workload"},
		Example: `
acorn workloadclasses`,
		SilenceUsage:      true,
		Short:             "List available WorkloadClasses",
		ValidArgsFunction: newCompletion(c.ClientFactory, workloadClassCompletion).complete,
	})
}

type WorkloadClass struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (a *WorkloadClass) Run(cmd *cobra.Command, args []string) error {
	c, err := a.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.WorkloadClass, a.Quiet, a.Output)

	if len(args) == 1 {
		wc, err := c.WorkloadClassGet(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		out.Write(wc)
		return out.Err()
	}

	wcs, err := c.WorkloadClassList(cmd.Context())
	if err != nil {
		return err
	}

	for _, wc := range wcs {
		if len(args) == 0 || slices.Contains(args, wc.Name) {
			out.Write(wc)
		}
	}

	return out.Err()
}
