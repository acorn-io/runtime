package publish

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/ports"
	"github.com/acorn-io/runtime/pkg/profiles"
	"github.com/acorn-io/z"
	"github.com/rancher/wrangler/pkg/name"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	customDomain  = "custom-domain"
	clusterDomain = "cluster-domain"
)

var (
	ErrInvalidPattern           = errors.New("endpoint pattern is invalid")
	ErrSegmentExceededMaxLength = errors.New("segment exceeded maximum length of 63 characters")
	ErrParsedEndpointIsNil      = errors.New("parsed endpoint pattern and recieved nil")
)

const dnsValidationPattern = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"

func ValidateEndpointPattern(pattern string) error {
	// Validate the Go Template
	endpoint, err := toHTTPEndpointHostname(pattern, "clusterdomain", "container", "app", "namespace")
	if err != nil {
		return err
	}

	// Validate the domain
	valid, err := regexp.MatchString(dnsValidationPattern, endpoint)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf(
			"%w: http-endpoint-pattern \"%v\" will look like \"%v\" which is not a valid domain, regex used for validation is: %v",
			ErrInvalidPattern,
			pattern,
			endpoint,
			dnsValidationPattern)
	}

	return nil
}

func truncate(s ...string) string {
	return name.SafeConcatName(s...)
}

func toHTTPEndpointHostname(pattern, domain, container, appName, appNamespace string) (string, error) {
	// This should not happen since the pattern in the config (passed to this through pattern) should
	// always be set to the default if the pattern is "". However,if it is not somehow, set it here.
	if pattern == "" {
		pattern = profiles.HttpEndpointPatternDefault
	}

	endpointOpts := struct {
		App           string
		Container     string
		Namespace     string
		Hash          string
		ClusterDomain string
	}{
		App:           appName,
		Container:     container,
		Namespace:     appNamespace,
		Hash:          hash(8, strings.Join([]string{container, appName, appNamespace}, ":")),
		ClusterDomain: strings.TrimPrefix(domain, "."),
	}

	var templateBuffer bytes.Buffer
	t := template.Must(template.New("").Funcs(map[string]any{
		"truncate":   truncate,
		"hashConcat": hashConcat,
	}).Parse(pattern))
	if err := t.Execute(&templateBuffer, endpointOpts); err != nil {
		return "", fmt.Errorf("%w %v: %v", ErrInvalidPattern, pattern, err)
	}

	endpoint := templateBuffer.String()
	if endpoint == "<nil>" || endpoint == "" {
		return "", ErrParsedEndpointIsNil
	}

	for _, segment := range strings.Split(endpoint, ".") {
		if len(segment) > 63 {
			return "", fmt.Errorf("%w: %v", ErrSegmentExceededMaxLength, segment)
		}
	}

	return templateBuffer.String(), nil
}

/*
hashConcat takes args, concatenate all the items except the last one, with a hash of
a concatenation of all items with ":".
*/
func hashConcat(limit int, args ...string) string {
	if len(args) < 2 {
		//Todo: this is to prevent runaway behavior in case it takes less than two parameters.
		//we don't have desired output for this but would rather return empty to prevent unexpected crash
		return ""
	}
	result := strings.Join(args, ":")

	return strings.Join(append(args[:len(args)-1], hash(limit, result)), "-")
}

func hash(limit int, s string) string {
	resultHash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(resultHash[:])[:limit]
}

type Target struct {
	Port    int32  `json:"port,omitempty"`
	Service string `json:"service,omitempty"`
}

func Ingress(req router.Request, svc *v1.ServiceInstance) (result []kclient.Object, _ error) {
	if svc.Spec.PublishMode == v1.PublishModeNone {
		return nil, nil
	}

	bindings := ports.ApplyBindings(svc.Spec.PublishMode, svc.Spec.Publish, ports.ByProtocol(svc.Spec.Ports, v1.ProtocolHTTP))

	if len(bindings) == 0 {
		return nil, nil
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	ingressClassName := cfg.IngressClassName
	if ingressClassName == nil {
		ingressClassName, err = IngressClassNameIfNoDefault(req.Ctx, req.Client)
		if err != nil {
			return nil, err
		}
	}

	var (
		// Separate rules for cluster domain and custom domain
		// This is needed to have separate ingress resources so that for custom domain, we can apply cert-manager setting to request certificate,
		// while keeping cluster domain certs as it is with acorn's built-in LE feature.
		customDomainRules    []networkingv1.IngressRule
		clusterDomainRules   []networkingv1.IngressRule
		customDomainTargets  = map[string]Target{}
		clusterDomainTargets = map[string]Target{}
	)

	for _, entry := range typed.Sorted(bindings.ByHostname()) {
		hostname := entry.Key
		ports := typed.MapSlice(entry.Value, func(p v1.PortDef) v1.PortDef {
			return p.Complete()
		})
		if hostname == "" {
			for i, port := range ports {
				targetName := svc.Name
				if i > 0 {
					targetName = name.SafeConcatName(targetName, fmt.Sprint(port.Port))
				}

				for _, domain := range cfg.ClusterDomains {
					hostname, err := toHTTPEndpointHostname(*cfg.HttpEndpointPattern, domain, targetName, svc.Spec.AppName, svc.Spec.AppNamespace)
					if err != nil {
						return nil, err
					}
					clusterDomainTargets[hostname] = Target{Port: port.TargetPort, Service: svc.Name}
					clusterDomainRules = append(clusterDomainRules, getIngressRule(svc, hostname, port.Port))
				}
			}
		} else {
			if len(ports) > 1 {
				return nil, fmt.Errorf("multiple ports bound to the same hostname [%s]", hostname)
			}
			customDomainTargets[hostname] = Target{Port: ports[0].TargetPort, Service: svc.Name}
			customDomainRules = append(customDomainRules, getIngressRule(svc, hostname, ports[0].Port))
		}
	}

	for _, rules := range []struct {
		rules  []networkingv1.IngressRule
		name   string
		target map[string]Target
	}{
		{rules: clusterDomainRules, name: clusterDomain, target: clusterDomainTargets},
		{rules: customDomainRules, name: customDomain, target: customDomainTargets},
	} {
		if len(rules.rules) == 0 {
			continue
		}
		// For custom domain, always use cert-manager to provision certificate.
		secrets, ingressTLS, ingressAnnotation, err := setupCertsForRules(req, svc, rules.rules, rules.name == customDomain, *cfg.CertManagerIssuer)
		if err != nil {
			return nil, err
		}

		targetJSON, err := json.Marshal(rules.target)
		if err != nil {
			return nil, err
		}

		proto := v1.PublishProtocolHTTP
		if len(ingressTLS) > 0 {
			proto = v1.PublishProtocolHTTPS
		}

		hostnameSeen := map[string]struct{}{}
		for _, rule := range rules.rules {
			if _, ok := hostnameSeen[rule.Host]; ok {
				continue
			}
			hostnameSeen[rule.Host] = struct{}{}
			svc.Status.Endpoints = append(svc.Status.Endpoints, v1.Endpoint{
				Address:         rule.Host,
				PublishProtocol: proto,
			})
		}

		ingress := &networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name.SafeConcatName(svc.Name, rules.name),
				Namespace: svc.Namespace,
				Labels:    svc.Spec.Labels,
				Annotations: labels.Merge(ingressAnnotation, map[string]string{
					labels.AcornTargets: string(targetJSON),
				}),
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: ingressClassName,
				Rules:            rules.rules,
				TLS:              ingressTLS,
			},
		}
		result = append(result, ingress)

		result = append(result, secrets...)
	}

	return
}

func setupCertManager(serviceName string, rules []networkingv1.IngressRule) []networkingv1.IngressTLS {
	var result []networkingv1.IngressTLS
	hostsSeen := map[string]bool{}
	for _, rule := range rules {
		if hostsSeen[rule.Host] {
			continue
		}
		hostsSeen[rule.Host] = true
		result = append(result, networkingv1.IngressTLS{
			Hosts:      []string{rule.Host},
			SecretName: name.SafeConcatName(serviceName, "cm-cert", strconv.Itoa(len(hostsSeen))),
		})
	}

	return result
}

// IngressClassNameIfNoDefault returns an ingress class name if there is exactly one IngressClass and it is not
// set as the default. We return a pointer here because "" is not a valid value for ingressClassName and will cause
// the ingress to fail.
func IngressClassNameIfNoDefault(ctx context.Context, client kclient.Client) (*string, error) {
	var ingressClasses networkingv1.IngressClassList
	if err := client.List(ctx, &ingressClasses); err != nil {
		return nil, err
	}
	if len(ingressClasses.Items) == 1 {
		ic := ingressClasses.Items[0]
		val := ic.Annotations["ingressclass.kubernetes.io/is-default-class"]
		if val != "true" {
			return &ic.Name, nil
		}
	}
	return nil, nil
}

func getIngressRule(svc *v1.ServiceInstance, host string, port int32) networkingv1.IngressRule {
	// strip possible port in host
	host, _, _ = strings.Cut(host, ":")

	if len(svc.Spec.Routes) > 0 {
		return routerRule(host, svc.Spec.Routes)
	}

	return networkingv1.IngressRule{
		Host: host,
		IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{
				Paths: []networkingv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: z.P(networkingv1.PathTypePrefix),
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: svc.Name,
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
