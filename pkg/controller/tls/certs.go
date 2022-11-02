package tls

import (
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ProvisionWildcardCert provisions a Let's Encrypt wildcard certificate for *.<domain>.on-acorn.io
func ProvisionWildcardCert(req router.Request, cfg *apiv1.Config, domain, token string) error {
	if !strings.EqualFold(*cfg.LetsEncrypt, "disabled") {
		logrus.Infof("Provisioning wildcard cert for %v", domain)
		// Ensure that we have a Let's Encrypt account ready
		leUser, err := ensureLEUser(req.Ctx, cfg, req.Client)
		if err != nil {
			return err
		}

		// Generate wildcard certificate for domain
		_, err = leUser.ensureWildcardCertificateSecret(req.Ctx, req.Client, *cfg.AcornDNSEndpoint, domain, token)
		if err != nil {
			return err
		}
	}

	return nil
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

// IgnoreWildcardCertSecret is a middleware that ignores the on-acorn.io wildcard cert secrets
func IgnoreWildcardCertSecret(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		sec := req.Object.(*corev1.Secret)
		if v, ok := sec.Annotations[labels.AcornDomain]; ok {
			if strings.HasSuffix(v, "on-acorn.io") {
				return nil
			}
		}
		return h.Handle(req, resp)
	})
}

// RenewCert handles the renewal of existing TLS certificates
func RenewCert(req router.Request, resp router.Response) error {
	logrus.Infof("Renewing TLS cert for %v", req.Key)
	return nil
}

// ProvisionCerts handles the provisioning of new TLS certificates for AppInstances
// Note: this does not actually provision the certificates, it just creates the empty secret
// which is picked up by the route handled by RenewCert above
func ProvisionCerts(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)

	appInstanceIDSegment := strings.SplitN(string(appInstance.GetUID()), "-", 2)[0]

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	leUser, err := ensureLEUser(req.Ctx, cfg, req.Client)

	// FIXME: use error list instead of exiting on first error?
	for _, pb := range appInstance.Spec.Ports {
		// Skip: name empty, not FQDN, already covered by on-acorn.io
		if pb.ServiceName == "" || len(validation.IsFullyQualifiedDomainName(&field.Path{}, pb.ServiceName)) > 0 || strings.HasSuffix(pb.ServiceName, "on-acorn.io") {
			continue
		}

		secretName := name.Limit(appInstance.GetName()+"-tls-"+pb.ServiceName, 63-len(appInstanceIDSegment)-1) + "-" + appInstanceIDSegment

		// Find existing secret if exists
		existingSecret := &corev1.Secret{}
		findSecretErr := req.Client.Get(req.Ctx, router.Key(appInstance.Namespace, secretName), existingSecret)
		if findSecretErr == nil {
			// Skip: TLS secret already exists, nothing to do here, renewal will be handled elsewhere
			continue
		} else if !apierrors.IsNotFound(findSecretErr) {
			// Not found is ok, we'll create the secret below.. other errors are bad
			logrus.Errorf("Error getting secret %s/%s: %v", appInstance.Namespace, secretName, findSecretErr)
			return findSecretErr
		}

		logrus.Infof("Provisioning TLS cert for %v in secret %s/%s", pb.ServiceName, appInstance.Namespace, secretName)
		cert, err := leUser.getCert(req.Ctx, pb.ServiceName)
		if err != nil {
			return fmt.Errorf("Error getting cert for %v: %v", pb.ServiceName, err)
		}

		newSec, err := leUser.certToSecret(cert, pb.ServiceName, appInstance.Namespace, secretName)
		if err != nil {
			return fmt.Errorf("Error converting cert to secret: %v", err)
		}

		if err := req.Client.Create(req.Ctx, newSec); err != nil {
			return fmt.Errorf("error creating TLS secret %s/%s: %v", appInstance.Namespace, secretName, err)
		}

	}

	return nil
}
