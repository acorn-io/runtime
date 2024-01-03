package cli

import (
	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	"github.com/spf13/cobra"
)

func NewSnapshotDelete(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SnapshotDelete{clientFactory: c.ClientFactory}, cobra.Command{
		Use:               "rm [flags] SNAPSHOT_NAME",
		SilenceUsage:      true,
		Short:             "Delete a snapshot",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, snapshotsCompletion).complete,
	})

	return cmd
}

type SnapshotDelete struct {
	clientFactory ClientFactory
}

func (sc *SnapshotDelete) Run(cmd *cobra.Command, args []string) error {
	cl, err := sc.clientFactory.CreateDefault()
	if err != nil {
		return err
	}

	return cl.SnapshotDelete(cmd.Context(), args[0])
}
