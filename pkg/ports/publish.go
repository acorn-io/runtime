package ports

import (
	"fmt"
	"sort"

	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
)

var clusterDomainHTTPDef = ListenDef{
	Protocol: v1.ProtocolHTTP,
}

func LinkService(app *v1.AppInstance, name string) string {
	if name == "" {
		return ""
	}

	for _, binding := range app.Spec.Links {
		if binding.Target == name {
			return binding.Service
		}
	}

	return ""
}

func IsLinked(app *v1.AppInstance, name string) bool {
	if name == "" {
		return false
	}

	for _, binding := range app.Spec.Links {
		if binding.Target == name {
			return true
		}
	}

	return false
}

func ByProtocol(ports []v1.PortDef, protocols ...v1.Protocol) (result []v1.PortDef) {
	for _, port := range ports {
		for _, proto := range protocols {
			if port.Complete().Protocol == proto {
				result = append(result, port)
				break
			}
		}
	}
	return result
}

type BoundPorts map[ListenDef][]v1.PortDef

func (b BoundPorts) ServicePorts() (result []corev1.ServicePort, err error) {
	for listen, ports := range b {
		if len(ports) > 1 {
			l := typed.MapSlice(ports, func(t v1.PortDef) string {
				return t.FormatString("")
			})
			return nil, fmt.Errorf("port [%d] is bound to [%d] ports %v, can only be bound to 1",
				listen.Port, len(ports), l)
		}
		port := ports[0].Complete()
		if listen.Port != 0 {
			port.Port = listen.Port
		}
		result = append(result, ToServicePort(port))
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return
}

func (b BoundPorts) ByHostname() map[string][]v1.PortDef {
	byHostname := map[string][]v1.PortDef{}
	for k, v := range b {
		byHostname[k.Hostname] = v
	}
	return byHostname
}

type ListenDef struct {
	// Hostname is empty if Protocol is not http
	Hostname string
	// Port if tcp or udp, for http this should always be 0
	Port     int32
	Protocol v1.Protocol
}

func listenDefFromPort(port v1.PortDef) ListenDef {
	def := ListenDef{
		Protocol: port.Protocol,
	}
	if port.Protocol == v1.ProtocolHTTP {
		def.Hostname = port.Hostname
	} else {
		def.Port = port.Complete().Port
	}
	return def
}

func PortPublishForService(serviceName string, bindings []v1.PortBinding) (result []v1.PortPublish) {
	for _, binding := range bindings {
		if serviceMatches(serviceName, binding) {
			result = append(result, v1.PortPublish{
				Port:       binding.Port,
				Protocol:   binding.Protocol,
				Hostname:   binding.Hostname,
				TargetPort: binding.TargetPort,
			})
		}
	}
	return
}

func ApplyBindings(mode v1.PublishMode, bindings []v1.PortPublish, ports []v1.PortDef) (result BoundPorts) {
	result = BoundPorts{}

	if mode == v1.PublishModeNone {
		return nil
	}

	for _, port := range ports {
		var (
			published bool
		)

		for _, binding := range bindings {
			if matches(binding, port) {
				published = true

				def := listenDefFromPort(port)
				if binding.Hostname != "" && port.Protocol == v1.ProtocolHTTP {
					def.Hostname = binding.Hostname
				} else if binding.Port != 0 && port.Protocol != v1.ProtocolHTTP {
					def.Port = binding.Port
				}
				result[def] = append(result[def], port)
			}
		}

		if !published && (mode == v1.PublishModeAll || port.Publish) {
			published = true
			def := listenDefFromPort(port)
			result[def] = append(result[def], port)
		}

		if published && port.Protocol == v1.ProtocolHTTP {
			found := false
			for _, existingPort := range result[clusterDomainHTTPDef] {
				if existingPort == port {
					found = true
					break
				}
			}
			if !found {
				result[clusterDomainHTTPDef] = append(result[clusterDomainHTTPDef], port)
			}
		}
	}

	return
}

func portMatches(binding v1.PortPublish, port v1.PortDef) bool {
	return binding.TargetPort == 0 || binding.TargetPort == port.Port
}

func serviceMatches(serviceName string, binding v1.PortBinding) bool {
	// deal with deprecated fields
	if binding.Expose && !binding.Publish {
		return false
	}
	return binding.TargetServiceName == "" || binding.TargetServiceName == serviceName
}

func protoMatches(binding v1.PortPublish, port v1.PortDef) bool {
	return binding.Protocol == "" ||
		binding.Protocol == port.Protocol
}

func matches(binding v1.PortPublish, port v1.PortDef) bool {
	port = port.Complete()
	binding = binding.Complete()

	return protoMatches(binding, port) &&
		portMatches(binding, port)
}

func collectPorts(seen map[int32][]int32, seenHostnames map[string]struct{}, ports []v1.PortDef, devMode bool) (result []v1.PortDef) {
	for _, port := range ports {
		if !devMode && port.Dev {
			continue
		}

		// Can't have any duplicate hostnames, so check for that first
		if _, exists := seenHostnames[port.Hostname]; port.Hostname != "" && exists {
			continue
		}

		// If port.Port is 0, that means only the TargetPort has been defined, and not the public-facing Port.
		// The public-facing Port will ultimately use the same number as the TargetPort, so we'll set it here
		// so that the logic is correct for the rest of this function.
		if port.Port == 0 {
			port.Port = port.TargetPort
		}

		if targets, ok := seen[port.Port]; ok {
			// Check for special case: the same port is exposed on multiple hostnames, so keep both.
			if port.Hostname != "" {
				for _, t := range targets {
					if t == port.TargetPort {
						// Same port and target port but different hostnames, so keep both
						seen[port.Port] = append(targets, port.TargetPort)
						seenHostnames[port.Hostname] = struct{}{}
						result = append(result, port)
						break
					}
				}
			}
			continue
		}

		seen[port.Port] = []int32{port.TargetPort}
		if port.Hostname != "" {
			seenHostnames[port.Hostname] = struct{}{}
		}
		result = append(result, port)
	}
	return
}

func FilterDevPorts(ports []v1.PortDef, devMode bool) (result []v1.PortDef) {
	for _, port := range ports {
		if port.Dev && !devMode {
			continue
		}
		result = append(result, port)
	}
	return
}

func CollectContainerPorts(container *v1.Container, devMode bool) (result []v1.PortDef) {
	// seen represents a mapping of public port numbers to a combination of hostname and target port
	seen := map[int32][]int32{}
	seenHostnames := map[string]struct{}{}

	result = append(result, collectPorts(seen, seenHostnames, container.Ports, devMode)...)
	for _, entry := range typed.Sorted(container.Sidecars) {
		result = append(result, collectPorts(seen, seenHostnames, entry.Value.Ports, devMode)...)
	}

	return
}
