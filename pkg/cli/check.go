package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/install/check"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/spf13/cobra"
)

func NewCheck() *cobra.Command {
	return cli.Command(&Check{}, cobra.Command{
		Use: "check",
		Example: `
acorn check`,
		SilenceUsage: true,
		Short:        "Check if the cluster is ready for Acorn",
	})
}

type Check struct {
	Quiet  bool   `usage:"No Results. Success or Failure only." short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
}

func (a *Check) Run(cmd *cobra.Command, args []string) error {
	checkresult := check.Check(
		check.CheckNodesReady,
		check.CheckRBAC,
	)

	failures := 0
	for _, r := range checkresult {
		if !r.Passed {
			failures++
		}
	}

	if !a.Quiet {
		out := table.NewWriter(tables.CheckResult, system.UserNamespace(), a.Quiet, a.Output)
		for _, r := range checkresult {
			out.Write(&r)
		}
		if err := out.Err(); err != nil {
			fmt.Println(err)
		}
	}

	if failures > 0 {
		return fmt.Errorf("Preflight Check FAILED: %d issues", failures)
	}

	fmt.Println("Preflight Check PASSED")

	return nil
}
