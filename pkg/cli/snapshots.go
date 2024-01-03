package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/acorn-io/runtime/pkg/cli/builder/table"
	"github.com/acorn-io/runtime/pkg/tables"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
)

func NewSnapshot(c CommandContext) *cobra.Command {
	cmd := cli.Command(&Snapshot{client: c.ClientFactory}, cobra.Command{
		Use:     "snapshot [flags] [SNAPSHOT_NAME...]",
		Aliases: []string{"snapshots", "s"},
		Example: `
acorn snapshot`,
		SilenceUsage: true,
		Short:        "Manage snapshots",
	})
	cmd.AddCommand(NewSnapshotCreate(c))
	cmd.AddCommand(NewSnapshotDelete(c))
	cmd.AddCommand(NewSnapshotRestore(c))

	return cmd
}

type Snapshot struct {
	Quiet  bool   `usage:"Output only names" short:"q"`
	Output string `usage:"Output format (json, yaml, {{gotemplate}})" short:"o"`
	client ClientFactory
}

func (s *Snapshot) Run(cmd *cobra.Command, args []string) error {
	c, err := s.client.CreateDefault()
	if err != nil {
		return err
	}

	out := table.NewWriter(tables.Snapshot, s.Quiet, s.Output)

	snapshots, err := c.SnapshotList(cmd.Context())
	if err != nil {
		return nil
	}

	for _, snapshot := range snapshots {
		if len(args) > 0 {
			if slices.Contains(args, snapshot.Name) {
				out.Write(&snapshot)
			}
		} else {
			out.Write(&snapshot)
		}
	}

	return out.Err()
}
