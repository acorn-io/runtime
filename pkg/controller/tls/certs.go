package tls

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ProvisionWildcardCert provisions a Let's Encrypt wildcard certificate for *.<domain>.on-acorn.io
func ProvisionWildcardCert(req router.Request, domain, token string) error {
	logrus.Debugf("Provisioning wildcard cert for %v", domain)
	// Ensure that we have a Let's Encrypt account ready
	leUser, err := ensureLEUser(req.Ctx, req.Client)
	if err != nil {
		return err
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
		return err
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
// Note: this does not actually provision the certificates, it just creates the empty secret
// which is picked up by the route handled by RenewCert above
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
		return err
	}

	// FIXME: use error list instead of exiting on first error
	for _, pb := range appInstance.Spec.Ports {
		// Skip: name empty, not FQDN, already covered by on-acorn.io
		if pb.ServiceName == "" || len(validation.IsFullyQualifiedDomainName(&field.Path{}, pb.ServiceName)) > 0 || strings.HasSuffix(pb.ServiceName, "on-acorn.io") {
			continue
		}

		secretName := name.Limit(appInstance.GetName()+"-tls-"+pb.ServiceName, 63-len(appInstanceIDSegment)-1) + "-" + appInstanceIDSegment

		return leUser.provisionCertIfNotExists(req.Ctx, req.Client, pb.ServiceName, appInstance.Namespace, secretName)

	}

	return nil
}

func (u *LEUser) provisionCertIfNotExists(ctx context.Context, client kclient.Client, domain string, namespace string, secretName string) error {
	// Find existing secret if exists
	existingSecret := &corev1.Secret{}
	findSecretErr := client.Get(ctx, router.Key(namespace, secretName), existingSecret)
	if findSecretErr == nil {
		// Skip: TLS secret already exists, nothing to do here, renewal will be handled elsewhere
		return nil
	} else if !apierrors.IsNotFound(findSecretErr) {
		// Not found is ok, we'll create the secret below.. other errors are bad
		logrus.Errorf("Error getting secret %s/%s: %v", namespace, secretName, findSecretErr)
		return findSecretErr
	}

	go func() {
		// Do not start a new challenge if we already have one in progress
		if !lockDomain(domain) {
			logrus.Debugf("not starting certificate renewal: %v: %s", ErrCertificateRequestInProgress, domain)
			return
		}
		defer unlockDomain(domain)

		logrus.Infof("Provisioning TLS cert for %v in secret %s/%s", domain, namespace, secretName)
		cert, err := u.getCert(ctx, domain)
		if err != nil {
			logrus.Errorf("Error getting cert for %v: %v", domain, err)
			return
		}

		newSec, err := u.certToSecret(cert, domain, namespace, secretName)
		if err != nil {
			logrus.Errorf("Error converting cert to secret: %v", err)
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
