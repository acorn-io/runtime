package cli

import (
	"fmt"

	cli "github.com/acorn-io/acorn/pkg/cli/builder"
	"github.com/acorn-io/acorn/pkg/cli/builder/table"
	"github.com/acorn-io/acorn/pkg/install"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/pterm/pterm"
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

	Image string `usage:"Override the image used for test deployments." short:"i"`
}

func (a *Check) Run(cmd *cobra.Command, args []string) error {

	checkOpts := install.CheckOptions{RuntimeImage: a.Image}
	checkresult := install.RunChecks(cmd.Context(), checkOpts,
		install.CheckRBAC,
		install.CheckNodesReady,
		install.CheckDefaultStorageClass,
		install.CheckIngressCapability,
		install.CheckExec,
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
		err := fmt.Errorf("%d checks failed", failures)
		pterm.Error.Println(err)
		return err
	}

	pterm.Success.Println("Checks PASSED")

	return nil
}
