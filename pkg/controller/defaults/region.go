package defaults

import (
	"context"

	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func addDefaultRegion(ctx context.Context, c client.Client, appInstance *internalv1.AppInstance) error {
	if appInstance.Spec.Region != "" {
		appInstance.Status.Defaults.Region = ""
		return nil
	}

	ns := new(corev1.Namespace)
	if err := c.Get(ctx, client.ObjectKey{Name: appInstance.Namespace}, ns); err != nil {
		return err
	}

	appInstance.Status.Defaults.Region = ns.Annotations[labels.AcornProjectDefaultRegion]
	if appInstance.Status.Defaults.Region == "" {
		appInstance.Status.Defaults.Region = ns.Annotations[labels.AcornCalculatedProjectDefaultRegion]
	}
	if appInstance.Status.Defaults.Region == "" {
		appInstance.Status.Defaults.Region = "local"
	}

	return nil
}
