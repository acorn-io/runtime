package dns

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/version"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Daemon struct {
	client kclient.Client
}

func NewDaemon(client kclient.Client) *Daemon {
	return &Daemon{
		client: client,
	}
}

// RenewAndSync will renew the cluster's AcornDNS domain and corresponding records.
// It sends each ingress's full record (fqdn, type, values). In addition to renewing the records, the DNS service will
// return "out of sync" records that either don't exist or have different values on the DNS service side. This function
// will cause the ingresses for such records to resync.
//
// Retries on an exponential backoff until successful
func (d *Daemon) RenewAndSync(ctx context.Context) {
	err := wait.ExponentialBackoffWithContext(ctx, wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2,
		Steps:    10,
		Cap:      300 * time.Second,
	}, func(ctx context.Context) (done bool, err error) {
		return d.renewAndSync(ctx)
	})
	if err != nil {
		logrus.Errorf("Couldn't complete RenewAndSync: %v", err)
	}
}

func (d *Daemon) renewAndSync(ctx context.Context) (bool, error) {
	cfg, err := config.Get(ctx, d.client)
	if err != nil {
		logrus.Errorf("Failed to get config: %v", err)
		return false, nil
	}

	if strings.EqualFold(*cfg.AcornDNS, "disabled") {
		logrus.Debugf("Acorn DNS is disabled, not attempting DNS renewal")
		return true, nil
	}

	logrus.Infof("Renewing and syncing AcornDNS...")

	dnsSecret, domain, token, err := d.getDNSSecret(ctx)
	if err != nil {
		logrus.Errorf("Failed to get DNS secret: %v", err)
		return false, nil
	}

	if err := d.syncIngress(ctx, domain, token, *cfg.AcornDNSEndpoint, dnsSecret); err != nil {
		logrus.Errorf("Failed to sync ingress: %v", err)
		return false, nil
	}

	logrus.Infof("Renewed and synced AcornDNS!")

	return true, nil
}

func (d *Daemon) getDNSSecret(ctx context.Context) (secret *corev1.Secret, domain string, token string, err error) {
	dnsSecret := &corev1.Secret{}
	err = d.client.Get(ctx, router.Key(system.Namespace, system.DNSSecretName), dnsSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return secret, domain, token, fmt.Errorf("DNS secret %v/%v not found, not proceeding with DNS renewal", system.Namespace, system.DNSSecretName)
		}
		return secret, domain, token, fmt.Errorf("problem getting DNS secret %v/%v: %v", system.Namespace, system.DNSSecretName, err)
	}

	domain = string(dnsSecret.Data["domain"])
	token = string(dnsSecret.Data["token"])
	if domain == "" || token == "" {
		return secret, domain, token, fmt.Errorf("DNS secret %v/%v exists but is missing domain (%v) or token", system.Namespace, system.DNSSecretName, domain)
	}
	return secret, domain, token, nil
}

func (d *Daemon) syncIngress(ctx context.Context, domain, token, acornDNSEndpoint string, secret *corev1.Secret) error {
	var ingress netv1.Ingress
	err := d.client.Get(ctx, router.Key(system.Namespace, system.IngressName), &ingress)
	if err != nil {
		return fmt.Errorf("failed to get %v for DNS renewal: %v", system.IngressName, err)
	}

	// Build the system.IngressName ingress into a list of RecordRequests
	recordRequests, _ := ToRecordRequestsAndHash(domain, &ingress)

	// Send the recordRequests to AcornDNS to renew and find any out of sync records
	dnsClient := NewClient()
	response, err := dnsClient.Renew(acornDNSEndpoint, domain, token, RenewRequest{Records: recordRequests, Version: version.Get().Tag})
	if err != nil {
		if IsDomainAuthError(err) {
			if clearErr := ClearDNSToken(ctx, d.client, secret); err != nil {
				err = errors.Join(fmt.Errorf("failed to clear DNS token: %v", clearErr))
			}
		}
		return err
	}

	// If there were any out of sync records, remove the hash from the ingress so it
	// gets reprocessed by the acorn-controller. This will cause any out of sync records
	// to be created or updated as necessary.
	if len(response.OutOfSyncRecords) > 0 {
		delete(ingress.Annotations, labels.AcornDNSHash)
		err = d.client.Update(ctx, &ingress)
		if err != nil {
			return fmt.Errorf("problem updating %v ingress: %v", ingress.Name, err)
		}
	}

	return nil
}
