package tls

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// ProvisionWildcardCert provisions a Let's Encrypt wildcard certificate for *.<domain>.oss-acorn.io
func ProvisionWildcardCert(req router.Request, resp router.Response, domain, token string) error {
	logrus.Debugf("Provisioning wildcard cert for %v", domain)
	// Ensure that we have a Let's Encrypt account ready
	leUser, err := ensureLEUser(req.Ctx, req.Client)
	if err != nil {
		logrus.Errorf("failed to get/create lets-encrypt account in ProvisionWildcardCert: %v", err)
		resp.RetryAfter(15 * time.Second)
		return nil
	}

	wildcardDomain := fmt.Sprintf("*.%s", strings.TrimPrefix(domain, "."))

	// Generate wildcard certificate for domain
	return leUser.provisionCertIfNotExists(req.Ctx, req.Client, wildcardDomain, system.Namespace, system.TLSSecretName)
}

// RequireSecretTypeTLS is a middleware that ensures that we only act on TLS-Type secrets
func RequireSecretTypeTLS(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		sec := req.Object.(*corev1.Secret)

		if sec.Type != corev1.SecretTypeTLS {
			return nil
		}

		return h.Handle(req, resp)
	})
}

// RenewCert handles the renewal of existing TLS certificates
func RenewCert(req router.Request, resp router.Response) error {
	sec := req.Object.(*corev1.Secret)

	leUser, err := ensureLEUser(req.Ctx, req.Client)
	if err != nil {
		logrus.Errorf("failed to get/create lets-encrypt account in RenewCert: %v", err)
		resp.RetryAfter(15 * time.Second)
		return nil
	}

	// Early exit if existing cert is still valid
	if !leUser.mustRenew(sec) {
		logrus.Debugf("Certificate for %v is still valid", sec.Name)
		return nil
	}

	domain := sec.Annotations[labels.AcornDomain]

	go func() {
		// Do not start a new challenge if we already have one in progress
		if !lockDomain(domain) {
			logrus.Debugf("not starting certificate renewal: %v: %s", ErrCertificateRequestInProgress, domain)
			return
		}
		defer unlockDomain(domain)

		logrus.Infof("Renewing TLS cert for %s", domain)

		// Get new certificate
		cert, err := leUser.getCert(req.Ctx, domain)
		if err != nil {
			logrus.Errorf("Error getting cert for %v: %v", domain, err)
			return
		}

		// Convert cert to secret
		newSec, err := leUser.certToSecret(cert, domain, sec.Namespace, sec.Name)
		if err != nil {
			logrus.Errorf("Error converting cert to secret: %v", err)
			return
		}

		// Update existing secret
		if err := req.Client.Update(req.Ctx, newSec); err != nil {
			logrus.Errorf("Error updating secret: %v", err)
			return
		}

		logrus.Infof("TLS secret %s/%s renewed for domain %s", newSec.Namespace, newSec.Name, domain)
	}()

	return nil
}

// ProvisionCerts handles the provisioning of new TLS certificates for AppInstances
func ProvisionCerts(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	// Early exit if Let's Encrypt is not enabled
	// Just to be on the safe side, we check for all possible allowed configuration values
	if strings.EqualFold(*cfg.LetsEncrypt, "disabled") {
		return nil
	}

	appInstance := req.Object.(*v1.AppInstance)

	appInstanceIDSegment := strings.SplitN(string(appInstance.GetUID()), "-", 2)[0]

	leUser, err := ensureLEUser(req.Ctx, req.Client)
	if err != nil {
		logrus.Errorf("failed to get/create lets-encrypt account in ProvisionCerts: %v", err)
		resp.RetryAfter(15 * time.Second)
		return nil
	}

	provisionedCerts := map[string]interface{}{}
	var errs []error

	for i, ep := range appInstance.Status.AppStatus.Endpoints {
		if ep.Protocol != "http" {
			continue
		}

		if err := prov(req, leUser, ep.Address, appInstance.Name, appInstanceIDSegment, appInstance.Namespace); err != nil {
			return err
		}
		provisionedCerts[ep.Address] = nil
		ep.PublishProtocol = v1.PublishProtocolHTTPS
		appInstance.Status.AppStatus.Endpoints[i] = ep
	}

	for _, pb := range appInstance.Spec.Publish {
		if _, ok := provisionedCerts[pb.Hostname]; ok {
			continue
		}
		if err := prov(req, leUser, pb.Hostname, appInstance.Name, appInstanceIDSegment, appInstance.Namespace); err != nil {
			errs = append(errs, err)
			continue
		}
		provisionedCerts[pb.Hostname] = nil
	}

	return utilerrors.NewAggregate(errs)
}

func prov(req router.Request, leUser *LEUser, domain, appname, segment, namespace string) error {
	if domain == "" || len(validation.IsFullyQualifiedDomainName(&field.Path{}, domain)) > 0 || strings.HasSuffix(domain, "oss-acorn.io") {
		logrus.Debugf("Skipping cert provisioning for %s", domain)
		return nil
	}
	secretName := name.Limit(appname+"-tls-"+domain, 63-len(segment)-1) + "-" + segment

	return leUser.provisionCertIfNotExists(req.Ctx, req.Client, domain, namespace, secretName)
}

// certFromSecret converts TLS secret data to a TLS certificate
func certFromSecret(secret corev1.Secret) (*x509.Certificate, error) {
	tlsPEM, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, fmt.Errorf("key tls.crt not found in secret %s/%s", secret.Namespace, secret.Name)
	}

	tlsCertBytes, _ := pem.Decode(tlsPEM)
	if tlsCertBytes == nil {
		return nil, fmt.Errorf("failed to parse Cert PEM stored in secret  %s/%s", secret.Namespace, secret.Name)
	}

	return x509.ParseCertificate(tlsCertBytes.Bytes)
}

// findExistingCert returns the first TLS secret that matches the given domain
func findExistingCertSecret(ctx context.Context, client kclient.Client, target string) (*corev1.Secret, error) {
	// Find existing certificate if exists
	existingTLSSecrets := &corev1.SecretList{}
	if err := client.List(ctx, existingTLSSecrets, &kclient.ListOptions{LabelSelector: klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	})}); err != nil {
		logrus.Errorf("Error listing existing TLS secrets: %v", err)
	}

	for _, sec := range existingTLSSecrets.Items {
		logrus.Debugf("Found existing TLS secret: %s/%s", sec.Namespace, sec.Name)
		if sec.Type != corev1.SecretTypeTLS {
			continue
		}
		domain, ok := sec.Annotations[labels.AcornDomain]
		if !ok {
			continue
		}

		if domain == target {
			cert, err := certFromSecret(sec)
			if err != nil {
				logrus.Errorf("Error parsing cert from secret: %v", err)
				continue
			}
			if err := cert.VerifyHostname(target); err == nil {
				return &sec, nil
			}
		}
	}
	return nil, nil
}

func (u *LEUser) provisionCertIfNotExists(ctx context.Context, client kclient.Client, domain string, namespace string, secretName string) error {
	// Find existing secret by expected name
	existingSecret := &corev1.Secret{}
	findSecretErr := client.Get(ctx, router.Key(namespace, secretName), existingSecret)
	var mustUpdate bool
	if findSecretErr == nil {
		if existingSecret.Annotations[labels.AcornDomain] == domain {
			// Skip: TLS secret already exists and matches the domain, nothing to do here, renewal will be handled elsewhere
			logrus.Debugf("No need to provision cert for domain %s, secret for that domain already exists: %s/%s", domain, existingSecret.Namespace, existingSecret.Name)
			return nil
		}
		// There's an existing secret with a different domain but matching the expected name, so we'll update it
		logrus.Debugf("Updating existing secret %s/%s with domain %s to %s", existingSecret.Namespace, existingSecret.Name, existingSecret.Annotations[labels.AcornDomain], domain)
		mustUpdate = true
	} else if !apierrors.IsNotFound(findSecretErr) {
		// Not found is ok, we'll create the secret below.. other errors are bad
		return fmt.Errorf("Error getting certificate secret %s/%s: %w", namespace, secretName, findSecretErr)
	}

	if !mustUpdate {
		// Let's see if we have some existing certificate that matches the domain
		existingSecret, err := findExistingCertSecret(ctx, client, domain)
		if err != nil {
			return err
		}
		if existingSecret != nil {
			// We found an existing certificate
			// 1. It's in the same namespace, so it will be picked up automatically
			if existingSecret.Namespace == namespace {
				logrus.Debugf("Found existing TLS secret %s/%s for domain %s", existingSecret.Namespace, existingSecret.Name, domain)
				// Skip: TLS secret already exists, nothing to do here, renewal will be handled elsewhere
				return nil
			}

			logrus.Debugf("Found existing TLS secret %s/%s for domain %s, copying to %s/%s", existingSecret.Namespace, existingSecret.Name, domain, namespace, secretName)

			// 2. Copy secret to new namespace
			copiedSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
					Labels: map[string]string{
						labels.AcornManaged: "true",
					},
					Annotations: map[string]string{
						labels.AcornDomain: domain,
					},
				},
				Data: existingSecret.Data,
				Type: existingSecret.Type,
			}

			if err := client.Create(ctx, copiedSecret); err != nil {
				return fmt.Errorf("Error creating TLS secret %s/%s: %w", copiedSecret.Namespace, copiedSecret.Name, err)
			}
			return nil
		}
	}

	go func() {
		// Do not start a new challenge if we already have one in progress
		if !lockDomain(domain) {
			logrus.Debugf("not starting certificate renewal: %v: %s", ErrCertificateRequestInProgress, domain)
			return
		}
		defer unlockDomain(domain)

		logrus.Infof("Provisioning TLS cert for %s in secret %s/%s", domain, namespace, secretName)
		cert, err := u.getCert(ctx, domain)
		if err != nil {
			logrus.Errorf("Error getting cert for %s: %v", domain, err)
			return
		}

		newSec, err := u.certToSecret(cert, domain, namespace, secretName)
		if err != nil {
			logrus.Errorf("Error converting cert to secret: %v", err)
			return
		}

		if mustUpdate {
			if err := client.Update(ctx, newSec); err != nil {
				logrus.Errorf("error updating TLS secret %s/%s: %v", namespace, secretName, err)
				return
			}
			logrus.Infof("TLS secret %s/%s updated for domain %s", namespace, secretName, domain)
			return
		}

		if err := client.Create(ctx, newSec); err != nil {
			logrus.Errorf("error creating TLS secret %s/%s: %v", namespace, secretName, err)
			return
		}

		logrus.Infof("TLS secret %s/%s created for domain %s", namespace, secretName, domain)
	}()

	return nil
}
