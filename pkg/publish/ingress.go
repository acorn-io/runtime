package publish

import (
	"encoding/json"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/ports"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func toPrefix(serviceName string, appInstance *v1.AppInstance) string {
	hostPrefix := serviceName + "." + appInstance.Name
	if serviceName == "default" {
		hostPrefix = appInstance.Name
	}
	if appInstance.Namespace != system.DefaultUserNamespace {
		hostPrefix += "." + appInstance.Namespace
	}
	return hostPrefix
}

type Target struct {
	Port    int32  `json:"port,omitempty"`
	Service string `json:"service,omitempty"`
}

func Ingress(req router.Request, app *v1.AppInstance) (result []kclient.Object, _ error) {
	if app.Spec.Stop != nil && *app.Spec.Stop {
		// remove all ingress
		return nil, nil
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	ingressClassName := cfg.IngressClassName

	ps, err := ports.NewForIngressPublish(app)
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
			targets = map[string]Target{}
		)

		for i, port := range ps.PortsForService(serviceName) {
			hostnames, ok := ps.Hostnames[port]
			if ok {
				for _, hostname := range hostnames {
					targets[hostname] = Target{Port: port.TargetPort, Service: serviceName}
					rules = append(rules, rule(hostname, serviceName, port.Port))
				}
			}
			if len(hostnames) == 0 {
				hostPrefix := toPrefix(serviceName, app)
				if i > 0 {
					hostPrefix = toPrefix(name.SafeConcatName(serviceName, fmt.Sprint(port.Port)), app)
				}
				for _, domain := range cfg.ClusterDomains {
					hostname := hostPrefix + domain
					hostnameMinusPort, _, _ := strings.Cut(hostname, ":")
					targets[hostname] = Target{Port: port.TargetPort, Service: serviceName}
					rules = append(rules, rule(hostnameMinusPort, serviceName, port.Port))
				}
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

		labelMap, annotations := ingressLabelsAndAnnotations(serviceName, string(targetJSON), app)
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

func ingressLabelsAndAnnotations(name, targetJSON string, appInstance *v1.AppInstance) (map[string]string, map[string]string) {
	var resourceType string
	var resourceLabels, resourceAnnotations map[string]string
	for k, v := range appInstance.Status.AppSpec.Containers {
		if k == name {
			resourceLabels = v.Labels
			resourceAnnotations = v.Annotations
			resourceType = v1.LabelTypeContainer
			break
		}
	}
	if resourceType == "" {
		for k, v := range appInstance.Status.AppSpec.Acorns {
			if k == name {
				resourceLabels = v.Labels
				resourceAnnotations = v.Annotations
				resourceType = v1.LabelTypeAcorn
				break
			}
		}
	}

	labelMap := labels.Managed(appInstance, labels.AcornServiceName, name)
	labelMap = labels.Merge(labelMap, labels.GatherScoped(name, resourceType, appInstance.Status.AppSpec.Labels,
		resourceLabels, appInstance.Spec.Labels))

	annotations := map[string]string{
		labels.AcornTargets: targetJSON,
	}
	annotations = labels.Merge(annotations, labels.GatherScoped(name, resourceType, appInstance.Status.AppSpec.Annotations,
		resourceAnnotations, appInstance.Spec.Annotations))
	return labelMap, annotations
}

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
