package ports

import (
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	RouterPortDef = v1.PortDef{
		Publish:    true,
		Port:       80,
		Protocol:   v1.ProtocolHTTP,
		TargetPort: 8080,
	}
)

func CopyServicePorts(ports []corev1.ServicePort) (result []corev1.ServicePort) {
	for _, port := range ports {
		port.NodePort = 0
		result = append(result, port)
	}
	return DedupPorts(result)
}

func DedupPorts(ports []corev1.ServicePort) (result []corev1.ServicePort) {
	byName := map[string]*corev1.ServicePort{}
	for _, port := range ports {
		existing, ok := byName[port.Name]
		if ok {
			if existing.AppProtocol == nil || *existing.AppProtocol == "" {
				existing.AppProtocol = port.AppProtocol
			}
			continue
		}
		result = append(result, port)
		byName[port.Name] = &result[len(result)-1]
	}
	return
}

func ToServicePorts(ports []v1.PortDef) []corev1.ServicePort {
	return DedupPorts(typed.MapSlice(ports, ToServicePort))
}

func ToServicePort(port v1.PortDef) corev1.ServicePort {
	port = port.Complete()
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
