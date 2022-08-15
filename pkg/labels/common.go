package labels

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"golang.org/x/exp/maps"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func AddCommonLabelsAndAnnotations(appInstance *v1.AppInstance, objects []kclient.Object) {
	for _, o := range objects {
		maps.Copy(o.GetAnnotations(), appInstance.Spec.CommonAnnotations)
	}
	for _, o := range objects {
		maps.Copy(o.GetLabels(), appInstance.Spec.CommonLabels)
	}
}
