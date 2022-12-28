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

var (
	ErrInvalidPattern           = errors.New("endpoint pattern is invalid")
	ErrSegmentExceededMaxLength = errors.New("segment exceeded maximum length of 63 characters")
	ErrParsedEndpointIsNil      = errors.New("parsed endpoint pattern and recieved nil")
)

const dnsValidationPattern = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"

func ValidateEndpointPattern(pattern string) error {
	// Validate the Go Template
	endpoint, err := toEndpoint(pattern, "clusterdomain", "container", "app", "namespace")
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

func toEndpoint(pattern, domain, container, appName, appNamespace string) (string, error) {
	// This should not happen since the pattern in the config (passed to this through pattern) should
	// always be set to the default if the pattern is "". However,if it is not somehow, set it here.
	if pattern == "" {
		pattern = config.DefaultHttpEndpointPattern
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
		Hash:          getAppHash(container, appName, appNamespace),
		ClusterDomain: strings.TrimPrefix(domain, "."),
	}

	var templateBuffer bytes.Buffer
	t := template.Must(template.New("").Parse(pattern))
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

func getAppHash(serviceName, appName, appNamespace string) string {
	var appInstanceIDSegment string
	var appInstanceIDSegmentByte [32]byte

	appInstanceIDSegment = serviceName + ":" + appName + ":" + appNamespace
	appInstanceIDSegmentByte = sha256.Sum256([]byte(appInstanceIDSegment))
	return hex.EncodeToString(appInstanceIDSegmentByte[:])[:12]
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
	if ingressClassName == nil {
		ingressClassName, err = IngressClassNameIfNoDefault(req.Ctx, req.Client)
		if err != nil {
			return nil, err
		}
	}

	ps, err := ports.NewForIngressPublish(app)
	if err != nil {
		return nil, err
	}

	rawPS, err := ports.New(app)
	if err != nil {
		return nil, err
	}

	// Look for Secrets in the app namespace that contain cert manager TLS certs
	tlsCerts, err := getCerts(req, app.Namespace)
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
			svcName := serviceName
			if i > 0 {
				svcName = name.SafeConcatName(serviceName, fmt.Sprint(port.Port))
			}
			for _, domain := range cfg.ClusterDomains {
				hostname, err := toEndpoint(*cfg.HttpEndpointPattern, domain, svcName, app.GetName(), app.GetNamespace())
				if err != nil {
					return nil, err
				}
				hostnameMinusPort, _, _ := strings.Cut(hostname, ":")
				targets[hostname] = Target{Port: port.TargetPort, Service: serviceName}
				rules = append(rules, rule(hostnameMinusPort, serviceName, port.Port))
			}
		}

		targetJSON, err := json.Marshal(targets)
		if err != nil {
			return nil, err
		}

		filteredTLSCerts := filterCertsForPublishedHosts(rules, tlsCerts)
		for i, tlsCert := range filteredTLSCerts {
			originalSecret := &corev1.Secret{}

			err := req.Get(originalSecret, tlsCert.SecretNamespace, tlsCert.SecretName)
			if err != nil {
				return nil, err
			}
			secretName := name.SafeConcatName(tlsCert.SecretName, app.Name, app.Namespace, string(originalSecret.UID)[:12])
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
			filteredTLSCerts[i].SecretName = secretName
			filteredTLSCerts[i].SecretNamespace = app.Status.Namespace
		}

		tlsIngress := getCertsForPublishedHosts(rules, filteredTLSCerts)
		labelMap, annotations := ingressLabelsAndAnnotations(serviceName, string(targetJSON), app, ps, rawPS)
		tlsIngress = setupCertManager(serviceName, annotations, rules, tlsIngress)

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

func setupCertManager(serviceName string, annotations map[string]string, rules []networkingv1.IngressRule, tls []networkingv1.IngressTLS) []networkingv1.IngressTLS {
	if (annotations["cert-manager.io/cluster-issuer"] == "" && annotations["cert-manager.io/issuer"] == "") ||
		len(tls) != 0 {
		// cert-manager is not being used, or we have TLS for this
		return tls
	}

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
// set as the default. We return a pointer here because "" is not a valid value for ingressClassName and will cause the
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

func routerIngressLabelsAndAnnotations(name, targetJSON string, app *v1.AppInstance, portSet, rawPS *ports.Set) (map[string]string, map[string]string) {
	labelMap := labels.Managed(app, labels.AcornServiceName, name)
	anns := map[string]string{labels.AcornTargets: targetJSON}

	// This is complicated, but we need to do this because an ingress can be for multiple containers, if both containers
	//have a port with the same serviceName. So, this logic finds all the containers an ingress is for and
	// gathers the labels/annotations from them.
	if ports, ok := portSet.Services[name]; ok {
		for port := range ports {
			for _, t := range rawPS.Ports[port] {
				labelMap = labels.Merge(labelMap, labels.GatherScoped(t.RouterName, v1.LabelTypeRouter,
					app.Status.AppSpec.Labels, app.Status.AppSpec.Routers[t.RouterName].Labels, app.Spec.Labels))
				anns = labels.Merge(anns, labels.GatherScoped(t.RouterName, v1.LabelTypeRouter,
					app.Status.AppSpec.Annotations, app.Status.AppSpec.Routers[t.RouterName].Annotations, app.Spec.Annotations))
			}
		}
	}

	return labelMap, anns
}

func ingressLabelsAndAnnotations(name, targetJSON string, app *v1.AppInstance, portSet, rawPS *ports.Set) (map[string]string, map[string]string) {
	labelMap := labels.Managed(app, labels.AcornServiceName, name)
	anns := map[string]string{labels.AcornTargets: targetJSON}

	// This is complicated, but we need to do this because an ingress can be for multiple containers, if both containers
	//have a port with the same serviceName. So, this logic finds all the containers an ingress is for and
	// gathers the labels/annotations from them.
	if ports, ok := portSet.Services[name]; ok {
		for port := range ports {
			for _, t := range rawPS.Ports[port] {
				labelMap = labels.Merge(labelMap, labels.GatherScoped(t.ContainerName, v1.LabelTypeContainer,
					app.Status.AppSpec.Labels, app.Status.AppSpec.Containers[t.ContainerName].Labels, app.Spec.Labels))
				anns = labels.Merge(anns, labels.GatherScoped(t.ContainerName, v1.LabelTypeContainer,
					app.Status.AppSpec.Annotations, app.Status.AppSpec.Containers[t.ContainerName].Annotations, app.Spec.Annotations))
			}
		}
	}

	return labelMap, anns
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
