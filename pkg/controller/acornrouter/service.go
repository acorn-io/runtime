package acornrouter

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/meta"
	name2 "github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func toService(appInstance *v1.AppInstance) []meta.Object {
	service := appdefinition.ToService(appInstance, appInstance.Name, v1.Container{Ports: appInstance.Spec.Ports})
	if service != nil {
		service, ptrService := toAcornService(appInstance, service)
		return []meta.Object{service, ptrService}
	}
	return nil
}

func toAcornService(app *v1.AppInstance, svc *corev1.Service) (*corev1.Service, *corev1.Service) {
	systemName := name2.SafeConcatName(svc.Name, app.Namespace, string(app.UID[:12]))
	ptrSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: app.Namespace,
			Labels:    toAcornLabels(svc.Labels),
		},
		Spec: corev1.ServiceSpec{
			Type:         corev1.ServiceTypeExternalName,
			ExternalName: systemName + "." + system.Namespace + "." + system.ClusterDomain,
		},
	}
	svc.Name = systemName
	svc.Namespace = system.Namespace
	svc.Labels = toAcornLabels(svc.Labels)
	svc.Spec.Selector = toAcornLabels(svc.Spec.Selector)
	svc.Spec.InternalTrafficPolicy = &[]corev1.ServiceInternalTrafficPolicyType{corev1.ServiceInternalTrafficPolicyLocal}[0]
	return svc, ptrSvc
}

func toAcornLabels(l map[string]string) map[string]string {
	result := map[string]string{}
	for k, v := range l {
		if k == labels.AcornContainerName {
			k = labels.AcornAcornName
		}
		result[k] = v
	}
	return result
}
