package ports

import (
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ToServicePort(port v1.PortDef) corev1.ServicePort {
	servicePort := corev1.ServicePort{
		Name:     strconv.Itoa(int(port.Port)),
		Protocol: corev1.ProtocolTCP,
		Port:     port.Port,
		TargetPort: intstr.IntOrString{
			IntVal: port.InternalPort,
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

func Dedup(ports []v1.PortDef) (result []v1.PortDef) {
	existing := map[string]bool{}
	for _, port := range ports {
		key := strconv.Itoa(int(port.Port)) + "/" + string(NormalizeProto(port.Protocol))
		if existing[key] {
			continue
		}
		existing[key] = true
		result = append(result, port)
	}
	return
}

func CollectPorts(container v1.Container) (result []v1.PortDef) {
	result = append(result, container.Ports...)

	for _, entry := range typed.Sorted(container.Sidecars) {
		result = append(result, CollectPorts(entry.Value)...)
	}

	return
}

func ProtosMatch(normalize bool, left v1.Protocol, right v1.Protocol) bool {
	if left == v1.ProtocolNone || right == v1.ProtocolNone {
		return false
	}
	if left == "" || right == "" {
		return true
	}
	if left == v1.ProtocolAll || right == v1.ProtocolAll {
		return true
	}
	if normalize {
		return NormalizeProto(left) == NormalizeProto(right)
	}
	return left == right
}

func NormalizeProto(proto v1.Protocol) v1.Protocol {
	switch proto {
	case v1.ProtocolHTTP:
		return v1.ProtocolTCP
	}
	return proto
}

func Layer4(ports []v1.PortDef) (result []v1.PortDef) {
	for _, port := range ports {
		if IsLayer4Port(port) {
			result = append(result, port)
		}
	}
	return
}

func PortsForIngress(portDefs []v1.PortDef, portBindings []v1.PortBinding, publishProtocols []v1.Protocol) (result []v1.PortDef) {
	for _, portDef := range portDefs {
		if !portDef.Expose || portDef.Protocol != v1.ProtocolHTTP {
			continue
		}
		matched := false
		for _, portBinding := range portBindings {
			if !portBinding.Publish {
				continue
			}
			if portBinding.TargetPort != portDef.Port {
				continue
			}
			if ProtosMatch(true, portBinding.Protocol, portDef.Protocol) {
				matched = true
				result = append(result, portDef)
				break
			}
		}

		if !matched {
			for _, protocol := range publishProtocols {
				if ProtosMatch(false, portDef.Protocol, protocol) {
					result = append(result, portDef)
				}
			}
		}
	}

	return
}

func RemapForBinding(publish bool, portDefs []v1.PortDef, portBindings []v1.PortBinding, publishProtocols []v1.Protocol) (result []v1.PortDef) {
	for _, portDef := range portDefs {
		if publish && !portDef.Expose {
			continue
		}
		matched := false
		for _, portBinding := range portBindings {
			if publish && !portBinding.Publish {
				continue
			}
			if portBinding.TargetPort != portDef.Port {
				continue
			}
			if ProtosMatch(true, portBinding.Protocol, portDef.Protocol) {
				matched = true
				internalPort := portDef.InternalPort
				if !publish {
					internalPort = portDef.Port
				}
				result = append(result, v1.PortDef{
					Port:         portBinding.Port,
					InternalPort: internalPort,
					Protocol:     portDef.Protocol,
					Expose:       portDef.Expose,
				})
				break
			}
		}

		if !matched {
			if !publish {
				publishProtocols = []v1.Protocol{v1.ProtocolAll}
			}
			for _, protocol := range publishProtocols {
				if ProtosMatch(false, portDef.Protocol, protocol) {
					internalPort := portDef.InternalPort
					if !publish {
						internalPort = portDef.Port
					}
					matched = true
					result = append(result, v1.PortDef{
						Port:         portDef.Port,
						InternalPort: internalPort,
						Protocol:     portDef.Protocol,
						Expose:       portDef.Expose,
					})
				}
			}
		}
	}

	return
}

func IsLayer4Port(port v1.PortDef) bool {
	switch port.Protocol {
	case v1.ProtocolUDP:
		return true
	case v1.ProtocolTCP:
		return true
	default:
		return false
	}
}
