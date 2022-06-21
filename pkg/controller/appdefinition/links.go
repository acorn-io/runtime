package appdefinition

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func addLinks(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(toLinks(appInstance)...)
}

func toLinks(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, link := range appInstance.Spec.Services {
		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      link.Target,
				Namespace: appInstance.Status.Namespace,
			},
			Spec: corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: fmt.Sprintf("%s.%s.%s", link.Service, appInstance.Namespace, system.ClusterDomain),
			},
		})
	}
	return
}
