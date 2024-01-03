package cli

import (
	"errors"

	cli "github.com/acorn-io/runtime/pkg/cli/builder"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

func NewSnapshotCreate(c CommandContext) *cobra.Command {
	cmd := cli.Command(&SnapshotCreate{client: c.ClientFactory}, cobra.Command{
		Use:               "create [flags] BOUND_VOLUME_NAME",
		SilenceUsage:      true,
		Short:             "Create a snapshot",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: newCompletion(c.ClientFactory, volumesCompletion).complete,
	})

	return cmd
}

type SnapshotCreate struct {
	Name   string `usage:"Give your snapshot a custom name" short:"n"`
	client ClientFactory
}

func (sc *SnapshotCreate) Run(cmd *cobra.Command, args []string) error {
	cl, err := sc.client.CreateDefault()
	if err != nil {
		return err
	}

	kc, err := cl.GetClient()
	if err != nil {
		return err
	}

	vol, err := cl.VolumeGet(cmd.Context(), args[0])
	if err != nil {
		return err
	}

	pvcs := &corev1.PersistentVolumeClaimList{}
	err = kc.List(cmd.Context(), pvcs, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged:    "true",
			labels.AcornAppName:    vol.Labels[labels.AcornAppName],
			labels.AcornPublicName: vol.Name,
		}),
	})
	if err != nil {
		return err
	}

	if len(pvcs.Items) == 0 || len(pvcs.Items) > 1 {
		return errors.New("pvc not found")
	}

	pvc := &pvcs.Items[0]

	if sc.Name != "" {
		pvc.Labels["custom-name"] = sc.Name
	}

	_, err = cl.SnapshotCreate(cmd.Context(), pvc)
	if err != nil {
		return err
	}

	return nil
}
