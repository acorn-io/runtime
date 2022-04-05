package appdefinition

import (
	"strconv"

	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/config"
	"github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/ibuildthecloud/herd/pkg/system"
	"github.com/rancher/wrangler/pkg/name"
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
			case v1.ProtocolHTTPS:
				fallthrough
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
				case v1.ProtocolHTTPS:
					fallthrough
				case v1.ProtocolHTTP:
					httpPorts[int(port.Port)] = port
				}
			}
		}
		if len(httpPorts) == 0 {
			continue
		}

		hostPrefix := containerName
		if appInstance.Namespace != system.DefaultUserNamespace {
			hostPrefix += "." + appInstance.Namespace
		}

		defaultPort, ok := httpPorts[80]
		if !ok {
			defaultPort, ok = httpPorts[443]
			if !ok {
				defaultPort = httpPorts[typed.SortedKeys(httpPorts)[0]]
			}
		}

		for _, entry := range typed.Sorted(httpPorts) {
			var (
				port  = entry.Value
				rules []networkingv1.IngressRule
			)
			for _, domain := range clusterDomains {
				if defaultPort.Port == port.Port {
					rules = append(rules, rule(hostPrefix+domain, containerName, port.Port))
				}
				rules = append(rules, rule(hostPrefix+domain+":"+strconv.Itoa(int(port.Port)), containerName, port.Port))
			}

			resp.Objects(&networkingv1.Ingress{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name.SafeConcatName(containerName, strconv.Itoa(int(port.Port))),
					Namespace: appInstance.Status.Namespace,
					Labels: containerLabels(appInstance, containerName,
						labels.HerdManaged, "true"),
				},
				Spec: networkingv1.IngressSpec{
					IngressClassName: ingressClassName,
					Rules:            rules,
				},
			})
		}
	}

	return nil
}
