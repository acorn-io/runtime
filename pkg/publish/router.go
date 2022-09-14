package publish

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Router(req router.Request, app *v1.AppInstance) (result []kclient.Object, _ error) {
	if app.Spec.Stop != nil && *app.Spec.Stop {
		// remove all ingress
		return nil, nil
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	ingressClassName := cfg.IngressClassName

	ps, err := ports.NewForRouterPublish(app)
	if err != nil {
		return nil, err
	}

	rawPS, err := ports.New(app)
	if err != nil {
		return nil, err
	}

	// Look for Secrets in the app namespace that contain cert manager TLS certs
	tlsCerts, err := getCerts(req, app.Labels[labels.AcornRootNamespace])
	if err != nil {
		return nil, err
	}

	for _, serviceName := range ps.ServiceNames() {
		var (
			rules   []networkingv1.IngressRule
			router  = app.Status.AppSpec.Routers[serviceName]
			targets = map[string]Target{}
		)

		for i, port := range ps.PortsForService(serviceName) {
			hostnames, ok := ps.Hostnames[port]
			if ok {
				for _, hostname := range hostnames {
					targets[hostname] = Target{Port: port.TargetPort, Service: serviceName}
					rules = append(rules, routerRule(hostname, router))
				}
			}
			hostPrefix := toPrefix(serviceName, app)
			if i > 0 {
				hostPrefix = toPrefix(name.SafeConcatName(serviceName, fmt.Sprint(port.Port)), app)
			}
			for _, domain := range cfg.ClusterDomains {
				hostname := hostPrefix + domain
				hostnameMinusPort, _, _ := strings.Cut(hostname, ":")
				targets[hostname] = Target{Port: port.TargetPort, Service: serviceName}
				rules = append(rules, routerRule(hostnameMinusPort, router))
			}
		}

		targetJSON, err := json.Marshal(targets)
		if err != nil {
			return nil, err
		}

		tlsIngress := getCertsForPublishedHosts(rules, tlsCerts)
		for i, ing := range tlsIngress {
			originalSecret := &corev1.Secret{}
			err := req.Get(originalSecret, app.Labels[labels.AcornRootNamespace], ing.SecretName)
			if err != nil {
				return nil, err
			}
			secretName := ing.SecretName + "-" + string(originalSecret.UID)[:12]
			result = append(result, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        secretName,
					Namespace:   app.Status.Namespace,
					Labels:      labels.Merge(originalSecret.Labels, labels.Managed(app)),
					Annotations: originalSecret.Annotations,
				},
				Type: corev1.SecretTypeTLS,
				Data: originalSecret.Data,
			})
			//Override the secret name to the copied name
			tlsIngress[i].SecretName = secretName
		}

		labelMap, annotations := routerIngressLabelsAndAnnotations(serviceName, string(targetJSON), app, ps, rawPS)
		result = append(result, &networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:        serviceName,
				Namespace:   app.Status.Namespace,
				Labels:      labelMap,
				Annotations: annotations,
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: ingressClassName,
				Rules:            rules,
				TLS:              tlsIngress,
			},
		})
	}

	return result, nil
}

func routerRule(host string, router v1.Router) networkingv1.IngressRule {
	rule := networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{},
		},
	}
	for _, route := range router.Routes {
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
