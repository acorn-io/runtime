package ports

import (
	"fmt"
	"sort"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/merr"
	"github.com/acorn-io/baaah/pkg/typed"
	"golang.org/x/exp/maps"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Target struct {
	ContainerName string
	AcornName     string
}

func (t Target) ServiceName() string {
	if t.ContainerName == "" {
		return t.AcornName
	}
	return t.ContainerName
}

type Set struct {
	Services  map[string]map[v1.PortDef]bool
	Ports     map[v1.PortDef][]Target
	Hostnames map[v1.PortDef][]string
}

func (p *Set) ServiceNames() []string {
	return typed.SortedKeys(p.Services)
}

func (p *Set) PortsForService(name string) []v1.PortDef {
	ports := maps.Keys(p.Services[name])
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Port == ports[j].Port {
			return ports[i].Protocol < ports[j].Protocol
		}
		return ports[i].Port < ports[j].Port
	})
	return ports
}

func (p *Set) AddPorts(target Target, ports ...v1.PortDef) {
	for _, port := range ports {
		port = port.Complete(target.ServiceName())
		s, ok := p.Services[port.ServiceName]
		if !ok {
			s = map[v1.PortDef]bool{}
			p.Services[port.ServiceName] = s
		}
		s[port] = true
		p.Ports[port] = append(p.Ports[port], target)
		sort.Slice(p.Ports[port], func(i, j int) bool {
			return p.Ports[port][i].ServiceName() < p.Ports[port][j].ServiceName()
		})
	}
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

	for _, port := range app.Status.AppSpec.Containers[name].Ports {
		if port.ServiceName != name && IsLinked(app, port.ServiceName) {
			return true
		}
	}

	for _, port := range app.Status.AppSpec.Acorns[name].Ports {
		if port.ServiceName != name && IsLinked(app, port.ServiceName) {
			return true
		}
	}

	return false
}

func (p *Set) GetContainerService(name string) string {
	if ports, ok := p.Services[name]; ok {
		for port := range ports {
			return p.Ports[port][0].ContainerName
		}
	}
	return ""
}

func (p *Set) IsContainerService(name string) bool {
	if ports, ok := p.Services[name]; ok {
		for port := range ports {
			return p.Ports[port][0].ContainerName != ""
		}
	}
	return false
}

func ForAcorn(app *v1.AppInstance, acornName string) (result []v1.PortBinding) {
	bound := map[v1.PortDef]bool{}
	for _, port := range app.Status.AppSpec.Acorns[acornName].Ports {
		pb := v1.PortBinding(port)
		pb.Expose = true
		if app.Spec.PublishMode == v1.PublishModeNone {
			pb.Publish = false
		}
		result = append(result, pb)
		for _, binding := range app.Spec.Ports {
			if !binding.Publish || !matches(binding, port) {
				continue
			}
			bound[port] = true
			result = append(result, v1.PortBinding{
				ServiceName:       binding.ServiceName,
				Port:              binding.Port,
				TargetPort:        port.TargetPort,
				TargetServiceName: port.TargetServiceName,
				Protocol:          port.Protocol,
				Publish:           true,
			})
		}
	}
	if app.Spec.PublishMode != v1.PublishModeNone {
		for _, port := range app.Status.AppSpec.Acorns[acornName].Ports {
			if bound[port] {
				continue
			}

			// ports with port.Publish=true will be added from the loop above
			if !port.Publish && app.Spec.PublishMode == v1.PublishModeAll {
				pb := v1.PortBinding(port)
				pb.Publish = true
				pb.Expose = false
				result = append(result, pb)
			}
		}
	}

	return
}

// Good example of combining whats in the acorn with whats been bound from the cli
// ps form New() is the port set from the acornfile by way of app.status.appspec.containers
func NewForIngressPublish(app *v1.AppInstance) (*Set, error) {
	ps, err := New(app)
	if err != nil {
		return nil, err
	}

	result := &Set{
		Services:  map[string]map[v1.PortDef]bool{},
		Ports:     map[v1.PortDef][]Target{},
		Hostnames: map[v1.PortDef][]string{},
	}

	bound := map[v1.PortDef]bool{}

	for port := range ps.Ports {
		if port.Protocol != v1.ProtocolHTTP || !ps.IsContainerService(port.ServiceName) {
			continue
		}

		for _, binding := range app.Spec.Ports {
			fullBinding := binding.Complete(app.Name)
			if !fullBinding.Publish || !matches(fullBinding, port) {
				continue
			}

			bound[port] = true

			if binding.ServiceName != "" {
				result.Hostnames[port] = append(result.Hostnames[port], binding.ServiceName)
			}
			result.AddPorts(Target{ContainerName: port.ServiceName}, port)
		}
	}

	if app.Spec.PublishMode != v1.PublishModeNone {
		for port := range ps.Ports {
			if bound[port] {
				continue
			}

			if port.Protocol != v1.ProtocolHTTP || !ps.IsContainerService(port.ServiceName) {
				continue
			}

			if port.Publish || app.Spec.PublishMode == v1.PublishModeAll {
				result.AddPorts(Target{ContainerName: port.ServiceName}, port)
			}
		}
	}

	return result, nil
}

func NewForAcornExpose(app *v1.AppInstance) (*Set, error) {
	ps, err := New(app)
	if err != nil {
		return nil, err
	}

	result := &Set{
		Services: map[string]map[v1.PortDef]bool{},
		Ports:    map[v1.PortDef][]Target{},
	}

	bound := map[v1.PortDef]bool{}

	for _, binding := range app.Spec.Ports {
		binding = binding.Complete(app.Name)

		var (
			boundProtocol    v1.Protocol
			boundServiceName = ""
		)
		for port := range ps.Ports {
			if !binding.Expose || !matches(binding, port) {
				continue
			}

			bound[port] = true

			if boundServiceName == "" {
				boundServiceName = port.ServiceName
			} else if boundServiceName != port.ServiceName {
				return nil, fmt.Errorf("ambigious port binding for [%d/%s] matches two services [%s, %s]", binding.TargetPort, binding.Protocol, boundServiceName, port.ServiceName)
			}

			if boundProtocol == "" {
				boundProtocol = port.Protocol
			} else if boundProtocol != port.Protocol {
				return nil, fmt.Errorf("ambigious port binding for [%d/%s] matches two protocols [%s, %s]", binding.TargetPort, binding.Protocol, boundProtocol, port.Protocol)
			}
		}

		if boundServiceName == "" {
			continue
		}
		result.AddPorts(Target{ContainerName: boundServiceName}, ToPortDef(binding, boundProtocol))
	}

	for port := range ps.Ports {
		if bound[port] {
			continue
		}

		if port.Expose {
			result.AddPorts(Target{ContainerName: port.ServiceName}, ToPortDef(v1.PortBinding{
				Port: port.Port,
			}.Complete(app.Name), port.Protocol))
		}
	}

	return result, nil
}

func matches(binding v1.PortBinding, port v1.PortDef) bool {
	if port.Protocol == v1.ProtocolHTTP {
		if binding.TargetPort != 0 && binding.TargetPort != port.Port {
			return false
		}
	} else if binding.TargetPort != port.Port {
		return false
	}
	if binding.Protocol != "" && binding.Protocol != port.Protocol {
		return false
	}
	if binding.TargetServiceName != "" && binding.TargetServiceName != port.ServiceName {
		return false
	}
	return true
}

func NewForContainerPublish(app *v1.AppInstance) (*Set, error) {
	ps, err := New(app)
	if err != nil {
		return nil, err
	}

	result := &Set{
		Services: map[string]map[v1.PortDef]bool{},
		Ports:    map[v1.PortDef][]Target{},
	}

	bound := map[v1.PortDef]bool{}

	for _, binding := range app.Spec.Ports {
		binding = binding.Complete(app.Name)

		for port := range ps.Ports {
			if port.Protocol != v1.ProtocolTCP && port.Protocol != v1.ProtocolUDP {
				continue
			}

			if !binding.Publish || !matches(binding, port) {
				continue
			}

			if ps.IsContainerService(port.ServiceName) {
				bound[port] = true
				port.Port = binding.Port
				result.AddPorts(Target{ContainerName: port.ServiceName}, port)
			}
		}
	}

	if app.Spec.PublishMode != v1.PublishModeNone {
		for port := range ps.Ports {
			if bound[port] {
				continue
			}

			if port.Protocol != v1.ProtocolTCP && port.Protocol != v1.ProtocolUDP {
				continue
			}

			if (port.Publish || app.Spec.PublishMode == v1.PublishModeAll) && ps.IsContainerService(port.ServiceName) {
				result.AddPorts(Target{ContainerName: port.ServiceName}, port)
			}
		}
	}

	return result, nil
}

func New(app *v1.AppInstance) (*Set, error) {
	result := &Set{
		Services: map[string]map[v1.PortDef]bool{},
		Ports:    map[v1.PortDef][]Target{},
	}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Containers) {
		containerName, container := entry.Key, entry.Value
		if IsLinked(app, containerName) {
			continue
		}

		result.AddPorts(Target{ContainerName: containerName}, container.Ports...)
		for _, sidecar := range typed.SortedValues(container.Sidecars) {
			result.AddPorts(Target{ContainerName: containerName}, sidecar.Ports...)
		}
	}

	for _, entry := range typed.Sorted(app.Status.AppSpec.Acorns) {
		acornName, acorn := entry.Key, entry.Value
		if IsLinked(app, acornName) {
			continue
		}
		result.AddPorts(Target{AcornName: acornName}, acorn.Ports...)
	}

	return result, validate(result)
}

func validate(m *Set) error {
	var errs []error
	for service, ports := range m.Services {
		var (
			foundContainer bool
			foundAcorn     bool
			oldTargetNames sets.String
			oldPort        v1.PortDef
		)
		for port := range ports {
			targetNames := sets.NewString()
			for _, target := range m.Ports[port] {
				if target.ContainerName != "" {
					targetNames.Insert(target.ContainerName)
					foundContainer = true
				}
				if target.AcornName != "" {
					targetNames.Insert(target.AcornName)
					foundAcorn = true
				}
			}
			if oldTargetNames == nil {
				oldTargetNames = targetNames
				oldPort = port
			} else if !oldTargetNames.Equal(targetNames) {
				errs = append(errs, fmt.Errorf("ports %s and %s on service %s do not share the same set of targets %v != %v",
					oldPort, port, service, oldTargetNames.List(), targetNames.List()))
			}
		}
		if foundContainer && foundAcorn {
			errs = append(errs, fmt.Errorf("service %s is addressing both containers and acorns, can only address one type", service))
		}
	}

	return merr.NewErrors(errs...)
}
