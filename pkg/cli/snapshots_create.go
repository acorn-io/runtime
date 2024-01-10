package cli

import (
	"errors"
	"strings"

	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
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
	Name          string `usage:"Give your snapshot a custom name" short:"n"`
	SnapshotClass string `usage:"Manually select the snapshot class used" short:"s"`
	client        ClientFactory
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

	apps, err := cl.AppList(cmd.Context())
	if err != nil {
		return err
	}

	appName := strings.TrimSuffix(vol.Labels[labels.AcornPublicName], "."+vol.Labels[labels.AcornVolumeName])
	var app v1.App
	for _, papp := range apps {
		if papp.Name == appName {
			app = papp
		}
	}

	if app.Status.Namespace == "" {
		return errors.New("unbound volumes cannot be snapshot")
	}

	pvcs := &corev1.PersistentVolumeClaimList{}
	err = kc.List(cmd.Context(), pvcs, &kclient.ListOptions{
		Namespace: app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged:    "true",
			labels.AcornAppName:    vol.Labels[labels.AcornAppName],
			labels.AcornPublicName: vol.Name,
		}),
	})
	if err != nil {
		return err
	}

	if len(pvcs.Items) == 0 {
		return errors.New("unbound volumes cannot be snapshot")
	}

	if len(pvcs.Items) > 1 {
		return errors.New("multiple persistent volume claims are associated with the requested volume")
	}

	pvc := &pvcs.Items[0]

	if sc.Name != "" {
		// the modifications to this PVC are not saved
		// so this additional label is just for internal use within SnapshotCreate
		// I did this as a lazy way of passing args into SnapshotCreate (no struct or additional params)
		pvc.Labels["acorn.io/custom-name"] = sc.Name
	}

	if sc.SnapshotClass != "" {
		pvc.Labels["acorn.io/snapshot-class"] = sc.SnapshotClass
	}

	_, err = cl.SnapshotCreate(cmd.Context(), pvc)
	if err != nil {
		return err
	}

	return nil
}
