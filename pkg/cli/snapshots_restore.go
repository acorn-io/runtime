package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewSnapshotRestore(c CommandContext) *cobra.Command {
	return cli.Command(&SnapshotRestore{client: c.ClientFactory}, cobra.Command{
		Use:               "restore [flags] SNAPSHOT_NAME VOLUME_NAME",
		SilenceUsage:      true,
		Short:             "Restore a snapshot to a new volume",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: newCompletion(c.ClientFactory, snapshotsCompletion).complete,
	})
}

type SnapshotRestore struct {
	client ClientFactory
}

func (sr *SnapshotRestore) Run(cmd *cobra.Command, args []string) error {
	cl, err := sr.client.CreateDefault()
	if err != nil {
		return err
	}

	return cl.SnapshotRestore(cmd.Context(), args[0], args[1])
}
