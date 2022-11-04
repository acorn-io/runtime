package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toServiceAccount(saName string, labelMap, annotations map[string]string, appInstance *v1.AppInstance) (result kclient.Object) {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        saName,
			Namespace:   appInstance.Status.Namespace,
			Labels:      labelMap,
			Annotations: annotations,
		},
	}
}
