package publish

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/tls"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TLSCert struct {
	Certificate     x509.Certificate `json:"certificate,omitempty"`
	SecretName      string           `json:"secret-name,omitempty"`
	SecretNamespace string           `json:"secret-namespace,omitempty"`
}

func (cert *TLSCert) certForThisDomain(name string) bool {
	if valid := cert.Certificate.VerifyHostname(name); valid == nil {
		return true
	}
	return false
}

func convertTLSSecretToTLSCert(secret corev1.Secret) (cert TLSCert, _ error) {
	tlsPEM, ok := secret.Data["tls.crt"]
	if !ok {
		return cert, fmt.Errorf("key tls.crt not found in secret %s", secret.Name)
	}

	tlsCertBytes, _ := pem.Decode(tlsPEM)
	if tlsCertBytes == nil {
		return cert, fmt.Errorf("failed to parse Cert PEM stored in secret %s", secret.Name)
	}

	tlsDataObj, err := x509.ParseCertificate(tlsCertBytes.Bytes)
	if err != nil {
		return cert, err
	}

	cert.SecretName = secret.Name
	cert.SecretNamespace = secret.Namespace
	cert.Certificate = *tlsDataObj

	return
}

// getCerts looks for Secrets in the app namespace that contain TLS certs
func getCerts(req router.Request, namespace string) ([]TLSCert, error) {
	var (
		result  []TLSCert
		secrets corev1.SecretList
	)

	err := req.List(&secrets, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return result, err
	}

	wildcardCertSecret := &corev1.Secret{}
	if err := req.Get(wildcardCertSecret, system.Namespace, system.TLSSecretName); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	if !apierrors.IsNotFound(err) {
		secrets.Items = append(secrets.Items, *wildcardCertSecret)
	}

	for _, secret := range secrets.Items {
		if secret.Type != corev1.SecretTypeTLS {
			continue
		}
		cert, err := convertTLSSecretToTLSCert(secret)
		if err != nil {
			logrus.Errorf("Error processing TLScertificate in secret %s/%s. Recieved %s", secret.Namespace, secret.Name, err)
			continue
		}

		result = append(result, cert)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].SecretName < result[j].SecretName
	})

	return result, nil
}

func copySecretsForCerts(req router.Request, svc *v1.ServiceInstance, filteredTLSCerts []TLSCert) (objs []client.Object, resultTLSCert []TLSCert, _ error) {
	for _, tlsCert := range filteredTLSCerts {
		originalSecret := &corev1.Secret{}

		err := req.Get(originalSecret, tlsCert.SecretNamespace, tlsCert.SecretName)
		if err != nil {
			return nil, nil, err
		}
		secretName := name.SafeConcatName(tlsCert.SecretName, svc.Name, string(originalSecret.UID)[:12])
		objs = append(objs, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: svc.Namespace,
				Labels:    labels.Merge(originalSecret.Labels, labels.ManagedByApp(svc.Spec.AppNamespace, svc.Spec.AppName)),
				Annotations: labels.Merge(originalSecret.Annotations, map[string]string{
					labels.AcornSecretSourceNamespace: originalSecret.Namespace,
					labels.AcornSecretSourceName:      originalSecret.Name,
				}),
			},
			Type: corev1.SecretTypeTLS,
			Data: originalSecret.Data,
		})

		//Override the secret name to the copied name
		tlsCert.SecretName = secretName
		tlsCert.SecretNamespace = svc.Namespace
		resultTLSCert = append(resultTLSCert, tlsCert)
	}

	return
}

func setupCertsForRules(req router.Request, svc *v1.ServiceInstance, rules []networkingv1.IngressRule, customDomain bool, defaultClusterIssuer string) ([]client.Object, []networkingv1.IngressTLS, map[string]string, error) {
	tlsCerts, err := getCerts(req, svc.Spec.AppNamespace)
	if err != nil {
		return nil, nil, nil, err
	}

	tlsCerts = getCertsMatchingRules(rules, tlsCerts)
	secrets, tlsCerts, err := copySecretsForCerts(req, svc, tlsCerts)
	if err != nil {
		return nil, nil, nil, err
	}

	annotations := maps.Clone(svc.Spec.Annotations)
	if annotations == nil {
		annotations = map[string]string{}
	}
	ingressTLS := getCertsForPublishedHosts(rules, tlsCerts)
	// In here, we want to setup cert-manager if:
	// 1. The user has specified a cert-manager issuer in the annotations on top of acorn
	// 2. The user has specified default cert-manager issuer in the settings, and there is no matching certs for this custom domain
	if svc.Spec.Annotations["cert-manager.io/cluster-issuer"] != "" || svc.Spec.Annotations["cert-manager.io/issuer"] != "" || (len(ingressTLS) == 0 && customDomain && defaultClusterIssuer != "") {
		if svc.Spec.Annotations["cert-manager.io/cluster-issuer"] == "" && svc.Spec.Annotations["cert-manager.io/issuer"] == "" {
			annotations["cert-manager.io/cluster-issuer"] = defaultClusterIssuer
		}
		ingressTLS = setupCertManager(svc.Name, rules)
	}

	// Best effort to wait for all domains to be ready, so we don't spam Let's Encrypt
	// with requests for domains where the DNS entry was not yet propagated
	hostsSeen := map[string]struct{}{}
	wg := sync.WaitGroup{}
	for _, rule := range rules {
		if _, ok := hostsSeen[rule.Host]; ok {
			continue
		}
		hostsSeen[rule.Host] = struct{}{}
		wg.Add(1)
		go func(host string) {
			err := tls.WaitForDomain(host, 5*time.Second, 6)
			if err != nil {
				logrus.Debugln(err)
			}
			wg.Done()
		}(rule.Host)
	}
	wg.Wait()
	return secrets, ingressTLS, annotations, nil
}

func getCertsMatchingRules(rules []networkingv1.IngressRule, certs []TLSCert) (filteredCerts []TLSCert) {
	for _, rule := range rules {
		for _, cert := range certs {
			if cert.certForThisDomain(rule.Host) {
				filteredCerts = append(filteredCerts, cert)
				break
			}
		}
	}
	return
}

func getCertsForPublishedHosts(rules []networkingv1.IngressRule, certs []TLSCert) (ingressTLS []networkingv1.IngressTLS) {
	certSecretToHostMapping := map[string][]string{}
	for _, rule := range rules {
		for _, cert := range certs {
			// Find the first cert and stop looking
			if cert.certForThisDomain(rule.Host) {
				certSecretToHostMapping[cert.SecretName] = append(certSecretToHostMapping[cert.SecretName], rule.Host)
				break
			}
		}
	}
	for _, entry := range typed.Sorted(certSecretToHostMapping) {
		secret, hosts := entry.Key, entry.Value
		ingressTLS = append(ingressTLS, networkingv1.IngressTLS{
			Hosts:      hosts,
			SecretName: secret,
		})
	}
	return
}
