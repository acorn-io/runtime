package defaults

import (
	"context"

	"github.com/acorn-io/acorn/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RegionGetterSetter interface {
	metav1.Object
	GetRegion() string
	SetDefaultRegion(string)
}

func AddDefaultRegion(ctx context.Context, c client.Client, obj RegionGetterSetter) error {
	if obj.GetRegion() == "" {
		ns := new(corev1.Namespace)
		if err := c.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, ns); err != nil {
			return err
		}

		region := ns.Annotations[labels.AcornProjectDefaultRegion]
		if region == "" {
			if r := ns.Annotations[labels.AcornCalculatedProjectDefaultRegion]; r != "" {
				region = r
			} else {
				region = "local"
			}
		}

		obj.SetDefaultRegion(region)
	}

	return nil
}
