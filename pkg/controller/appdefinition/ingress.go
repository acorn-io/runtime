package appdefinition

import (
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func rule(host, serviceName string, port int32) networkingv1.IngressRule {
	return networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: &[]networkingv1.PathType{networkingv1.PathTypePrefix}[0],
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: serviceName,
								Port: networkingv1.ServiceBackendPort{
									Number: port,
								},
							},
						},
					},
				},
			},
		},
	}
}

func addIngress(appInstance *v1.AppInstance, req router.Request, resp router.Response) error {
	if appInstance.Spec.Stop != nil && *appInstance.Spec.Stop {
		// remove all ingress
		return nil
	}

	cfg, err := config.Get(req.Client)
	if err != nil {
		return err
	}
	var ingressClassName *string
	if cfg.IngressClassName != "" {
		ingressClassName = &cfg.IngressClassName
	}

	clusterDomains := cfg.ClusterDomains
	if len(clusterDomains) == 0 {
		clusterDomains = []string{".localhost"}
	}

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Containers) {
		containerName := entry.Key
		httpPorts := map[int]v1.Port{}
		for _, port := range entry.Value.Ports {
			if !port.Publish {
				continue
			}
			switch port.Protocol {
			case v1.ProtocolHTTP:
				httpPorts[int(port.Port)] = port
			}
		}
		for _, sidecar := range entry.Value.Sidecars {
			for _, port := range sidecar.Ports {
				if !port.Publish {
					continue
				}
				switch port.Protocol {
				case v1.ProtocolHTTP:
					httpPorts[int(port.Port)] = port
				}
			}
		}
		if len(httpPorts) == 0 {
			continue
		}

		hostPrefix := containerName + "." + appInstance.Name
		if containerName == "default" {
			hostPrefix = appInstance.Name
		}
		if appInstance.Namespace != system.DefaultUserNamespace {
			hostPrefix += "." + appInstance.Namespace
		}

		defaultPort, ok := httpPorts[80]
		if !ok {
			defaultPort = httpPorts[typed.SortedKeys(httpPorts)[0]]
		}

		var (
			rules []networkingv1.IngressRule
			hosts []string
		)

		for _, binding := range appInstance.Spec.Endpoints {
			if binding.Target == containerName {
				hosts = append(hosts, binding.Hostname)
				rules = append(rules, rule(binding.Hostname, containerName, defaultPort.Port))
			}
		}

		addClusterDomains := len(hosts) == 0

		for _, domain := range clusterDomains {
			if addClusterDomains {
				hosts = append(hosts, hostPrefix+domain)
			}
			rules = append(rules, rule(hostPrefix+domain, containerName, defaultPort.Port))
		}

		resp.Objects(&networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      containerName,
				Namespace: appInstance.Status.Namespace,
				Labels:    containerLabels(appInstance, containerName),
				Annotations: map[string]string{
					labels.AcornHostnames:     strings.Join(hosts, ","),
					labels.AcornPortNumber:    strconv.Itoa(int(defaultPort.ContainerPort)),
					labels.AcornContainerName: containerName,
				},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: ingressClassName,
				Rules:            rules,
			},
		})
	}

	return nil
}
