package appdefinition

import (
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	kubernetesTLSSecretType         = "kubernetes.io/tls"
	certManagerCommonNameAnnotation = "cert-manager.io/common-name"
	certManagerDNSNamesAnnotation   = "cert-manager.io/alt-names"
)

type TLSCert struct {
	CommonName string          `json:"common-name,omitempty"`
	SANS       map[string]bool `json:"names,omitempty"`
	SecretName string          `json:"secret-name,omitempty"`
}

func (cert *TLSCert) certForThisDomain(name string) bool {
	if t, ok := cert.SANS[name]; ok {
		return t
	}
	return false
}

func (cert *TLSCert) sansList() (sans []string) {
	for k := range cert.SANS {
		sans = append(sans, k)
	}
	return
}

func getCerts(namespace string, req router.Request) ([]*TLSCert, error) {
	result := []*TLSCert{}

	var secrets corev1.SecretList
	err := req.Client.List(&secrets, &meta.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return result, err
	}

	if len(secrets.Items) > 0 {
		for _, secret := range secrets.Items {
			if secret.Type != kubernetesTLSSecretType {
				continue
			}
			cert := &TLSCert{
				SANS: make(map[string]bool),
			}

			name, ok := secret.Annotations[certManagerCommonNameAnnotation]
			if !ok {
				continue
			}

			cert.CommonName = name
			cert.SecretName = secret.Name
			cert.SANS[name] = true

			if alts, ok := secret.Annotations[certManagerDNSNamesAnnotation]; ok {
				for _, dnsName := range strings.Split(alts, ",") {
					cert.SANS[dnsName] = true
				}
			}

			result = append(result, cert)
		}
	}
	return result, nil
}

func getCertsForPublishedHosts(rules []networkingv1.IngressRule, certs []*TLSCert) (ingressTLS []networkingv1.IngressTLS) {
	for _, rule := range rules {
		for _, cert := range certs {
			if cert.certForThisDomain(rule.Host) {
				ingressTLS = append(ingressTLS, networkingv1.IngressTLS{
					Hosts:      cert.sansList(),
					SecretName: cert.SecretName,
				})
			}
		}
	}
	return
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

func toPrefix(containerName string, appInstance *v1.AppInstance) string {
	hostPrefix := containerName + "." + appInstance.Name
	if containerName == "default" {
		hostPrefix = appInstance.Name
	}
	if appInstance.Namespace != system.DefaultUserNamespace {
		hostPrefix += "." + appInstance.Namespace
	}
	return hostPrefix
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

	// Look for Secrets in the app namespace that contain cert manager TLS certs
	tlsCerts, err := getCerts(appInstance.Namespace, req)
	if err != nil {
		return err
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

		hostPrefix := toPrefix(containerName, appInstance)

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
			for _, alias := range entry.Value.Aliases {
				aliasPrefix := toPrefix(alias.Name, appInstance)
				if addClusterDomains {
					hosts = append(hosts, aliasPrefix+domain)
				}
				rules = append(rules, rule(aliasPrefix+domain, alias.Name, defaultPort.Port))
			}
		}

		tlsIngress := getCertsForPublishedHosts(rules, tlsCerts)
		for i, ing := range tlsIngress {
			originalSecret := &corev1.Secret{}
			err := req.Client.Get(originalSecret, ing.SecretName, nil)
			if err != nil {
				return err
			}
			secretName := ing.SecretName + "-" + string(originalSecret.UID)[:8]
			resp.Objects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        secretName,
					Namespace:   appInstance.Status.Namespace,
					Labels:      labelsForSecret(originalSecret.Name, appInstance),
					Annotations: originalSecret.Annotations,
				},
				Type: corev1.SecretTypeTLS,
				Data: originalSecret.Data,
			})
			//Override the secret name to the copied name
			tlsIngress[i].SecretName = secretName
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
				TLS:              tlsIngress,
			},
		})
	}

	return nil
}
