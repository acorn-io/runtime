package publish

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ServiceLoadBalancer(svc *v1.ServiceInstance) (result []kclient.Object, _ error) {
	if svc.Spec.PublishMode == v1.PublishModeNone {
		return nil, nil
	}

	bindings := ports.ApplyBindings(svc.Spec.PublishMode, svc.Spec.Publish,
		ports.ByProtocol(svc.Spec.Ports, v1.ProtocolTCP, v1.ProtocolUDP))

	if len(bindings) == 0 {
		return nil, nil
	}

	selectorLabels := svc.Spec.ContainerLabels
	if svc.Spec.Container != "" {
		selectorLabels = map[string]string{
			labels.AcornContainerName: svc.Spec.Container,
		}
	}

	if len(selectorLabels) == 0 {
		return nil, nil
	}

	servicePorts, err := bindings.ServicePorts()
	if err != nil {
		return nil, err
	}

	result = append(result, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.SafeConcatName(svc.Name, "publish", svc.ShortID()),
			Namespace: svc.Namespace,
			Labels: labels.Merge(svc.Spec.Labels, map[string]string{
				labels.AcornServicePublish: "true",
			}),
			Annotations: svc.Spec.Annotations,
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: labels.Merge(labels.ManagedByApp(svc.Spec.AppNamespace, svc.Spec.AppName), selectorLabels),
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	})

	return result, nil
}
