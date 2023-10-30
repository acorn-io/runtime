package publish

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ServiceLoadBalancer(req router.Request, svc *v1.ServiceInstance) (result []kclient.Object, _ error) {
	if svc.Spec.PublishMode == v1.PublishModeNone {
		return nil, nil
	}

	bindings := ports.ApplyBindings(svc.Spec.PublishMode, svc.Spec.Publish,
		ports.ByProtocol(svc.Spec.Ports, true, v1.ProtocolTCP, v1.ProtocolUDP))

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

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	if svc.Spec.Annotations == nil {
		svc.Spec.Annotations = map[string]string{}
	}

	for _, annotation := range cfg.ServiceLBAnnotations {
		key, value, found := strings.Cut(annotation, "=")
		// This shouldn't happen due to the same validation in the CLI side. Check anyway.
		if !found {
			return nil, fmt.Errorf("invalid annotation %s set in Config, must be in the form of key=value", annotation)
		}
		svc.Spec.Annotations[key] = value
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
			Annotations: typed.Concat(svc.Spec.Annotations, map[string]string{
				labels.AcornConfigHashAnnotation: svc.Annotations[labels.AcornConfigHashAnnotation],
			}),
		},
		Spec: corev1.ServiceSpec{
			Ports:    servicePorts,
			Selector: labels.Merge(labels.ManagedByApp(svc.Spec.AppNamespace, svc.Spec.AppName), selectorLabels),
			Type:     corev1.ServiceTypeLoadBalancer,
		},
	})

	return result, nil
}
