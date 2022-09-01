package ports

import (
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ToPodLabels(app *v1.AppInstance, containerName string) map[string]string {
	container := app.Status.AppSpec.Containers[containerName]
	ports := container.Ports
	for _, sidecar := range container.Sidecars {
		ports = append(ports, sidecar.Ports...)
	}
	return ToSelector(app, typed.MapSlice(ports, func(p v1.PortDef) v1.PortDef {
		return p.Complete(containerName)
	}))
}

func ToSelector(app *v1.AppInstance, ports []v1.PortDef) map[string]string {
	result := labels.Managed(app)
	for _, port := range ports {
		result[labels.AcornServiceNamePrefix+port.ServiceName] = "true"
		result[fmt.Sprintf("%s%d", labels.AcornPortNumberPrefix, port.TargetPort)] = "true"
	}

	return result
}

func ToPortDef(binding v1.PortBinding, protocol v1.Protocol) v1.PortDef {
	result := v1.PortDef{
		Port:       binding.Port,
		TargetPort: binding.TargetPort,
		Protocol:   protocol,
	}
	return result.Complete(binding.ServiceName)
}

func ToServicePort(port v1.PortDef) corev1.ServicePort {
	servicePort := corev1.ServicePort{
		Name:     strconv.Itoa(int(port.Port)),
		Protocol: corev1.ProtocolTCP,
		Port:     port.Port,
		TargetPort: intstr.IntOrString{
			IntVal: port.TargetPort,
		},
	}
	switch port.Protocol {
	case v1.ProtocolTCP:
	case v1.ProtocolUDP:
		servicePort.Protocol = corev1.ProtocolUDP
	case v1.ProtocolHTTP:
		str := strings.ToUpper(string(port.Protocol))
		servicePort.AppProtocol = &str
	}
	return servicePort
}

func NormalizeProto(proto v1.Protocol) v1.Protocol {
	switch proto {
	case v1.ProtocolHTTP:
		return v1.ProtocolTCP
	}
	return proto
}

func ToContainerServices(app *v1.AppInstance, publish bool, namespace string, portSet *Set) (result []kclient.Object) {
	for _, serviceName := range portSet.ServiceNames() {
		if !portSet.IsContainerService(serviceName) {
			continue
		}
		servicePorts := portSet.PortsForService(serviceName)
		if len(servicePorts) == 0 {
			continue
		}
		resourceName := serviceName
		serviceType := corev1.ServiceTypeClusterIP
		if publish {
			resourceName = name.SafeConcatName(resourceName, "publish", app.ShortID())
			serviceType = corev1.ServiceTypeLoadBalancer
		}
		extraLabels := []string{
			labels.AcornServiceName, serviceName,
			labels.AcornContainerName, portSet.GetContainerService(serviceName),
		}
		if publish {
			extraLabels = append(extraLabels, labels.AcornServicePublish, "true")
		}

		labelMap := labels.Managed(app, extraLabels...)
		anns := make(map[string]string)
		// This is complicated, but we need to do this because a service can be selecting multiple containers, if both
		// containers have a port with the same serviceName. So, this logic finds all the containers a service is selecting and
		// gathers the labels/annotations from them.
		if ports, ok := portSet.Services[serviceName]; ok {
			for port := range ports {
				for _, t := range portSet.Ports[port] {
					labelMap = labels.Merge(labelMap, labels.GatherScoped(t.ContainerName, v1.LabelTypeContainer, app.Status.AppSpec.Labels,
						app.Status.AppSpec.Containers[t.ContainerName].Labels, app.Spec.Labels))
					anns = labels.Merge(anns, labels.GatherScoped(t.ContainerName, v1.LabelTypeContainer, app.Status.AppSpec.Annotations,
						app.Status.AppSpec.Containers[t.ContainerName].Annotations, app.Spec.Annotations))
				}
			}
		}

		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        resourceName,
				Namespace:   namespace,
				Labels:      labelMap,
				Annotations: anns,
			},
			Spec: corev1.ServiceSpec{
				Ports:    typed.MapSlice(servicePorts, ToServicePort),
				Selector: ToSelector(app, servicePorts),
				Type:     serviceType,
			},
		})
	}
	return
}
