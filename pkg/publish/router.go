package publish

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func routerRule(host string, routes []v1.Route) networkingv1.IngressRule {
	rule := networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{},
		},
	}
	for _, route := range routes {
		if route.Path == "" || route.TargetServiceName == "" {
			continue
		}
		pathType := networkingv1.PathTypePrefix
		if route.PathType == v1.PathTypeExact {
			pathType = networkingv1.PathTypeExact
		}
		port := route.TargetPort
		if port == 0 {
			port = 80
		}
		rule.IngressRuleValue.HTTP.Paths = append(rule.IngressRuleValue.HTTP.Paths, networkingv1.HTTPIngressPath{
			Path:     route.Path,
			PathType: &pathType,
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: route.TargetServiceName,
					Port: networkingv1.ServiceBackendPort{
						Number: int32(port),
					},
				},
			},
		})
	}
	return rule
}
