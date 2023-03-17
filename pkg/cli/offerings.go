package cli

import (
	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewOfferings(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Offerings{}, cobra.Command{
		Use:     "offerings [flags] command",
		Aliases: []string{"offering", "o"},
		Example: `
acorn offerings`,
		Short: "Show infrastructure offerings",
	})
	cmd.AddCommand(NewVolumeClasses(c), NewComputeClasses(c), NewRegions(c))
	return cmd
}

type Offerings struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (o *Offerings) Run(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
