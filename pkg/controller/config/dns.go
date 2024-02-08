package config

import (
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/uncached"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/controller/tls"
	"github.com/acorn-io/runtime/pkg/dns"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewDNSConfigHandler() router.Handler {
	return &configHandler{
		dns: dns.NewClient(),
	}
}

type configHandler struct {
	dns dns.Client
}

// Handle communicates with the Acorn DNS service based on the whether the acorn-dns feature is enabled in the cluster.
// If it is enabled (explicitly or implicitly via "auto"), this will ensure a domain and token have been reserved and
// store in a secret.
// If it is disabled, it ensures existing records for the domain have been purged from the AcornDNS service.
func (h *configHandler) Handle(req router.Request, resp router.Response) error {
	cfg, err := config.UnmarshalAndComplete(req.Ctx, req.Object.(*corev1.ConfigMap), req.Client)
	if err != nil {
		return err
	}

	dnsSecret := &corev1.Secret{}
	err = req.Client.Get(req.Ctx, router.Key(system.Namespace, system.DNSSecretName), dnsSecret)
	if kclient.IgnoreNotFound(err) != nil {
		return err
	}
	domain := string(dnsSecret.Data["domain"])
	token := string(dnsSecret.Data["token"])

	state, err := purgeRecordsIfDisabling(req, domain, cfg, dnsSecret, token, h.dns)
	if err != nil {
		return err
	}

	if !strings.EqualFold(*cfg.AcornDNS, "disabled") && (domain == "" || token == "") {
		if domain != "" {
			logrus.Infof("Clearing AcornDNS domain  %v", domain)
		}
		domain, token, err = h.dns.ReserveDomain(*cfg.AcornDNSEndpoint)
		if err != nil {
			return fmt.Errorf("problem reserving domain: %w", err)
		}
		logrus.Infof("Obtained AcornDNS domain: %v", domain)
	}

	if dnsSecret.Name == "" {
		// Secret doesn't exist. Create it
		return req.Client.Create(req.Ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      system.DNSSecretName,
				Namespace: system.Namespace,
				Annotations: map[string]string{
					labels.AcornDNSState: state,
				},
				Labels: map[string]string{
					labels.AcornManaged: "true",
				},
			},
			Data: map[string][]byte{"domain": []byte(domain), "token": []byte(token)},
		})
	}

	// Secret exists. Update it
	sec := &corev1.Secret{}
	err = req.Client.Get(req.Ctx, kclient.ObjectKey{
		Name:      system.DNSSecretName,
		Namespace: system.Namespace,
	}, uncached.Get(sec))
	if err != nil {
		return err
	}

	if sec.Annotations[labels.AcornDNSState] != state ||
		string(sec.Data["domain"]) != domain ||
		string(sec.Data["token"]) != token {
		sec.Annotations[labels.AcornDNSState] = state
		sec.Data = map[string][]byte{"domain": []byte(domain), "token": []byte(token)}
		if err := req.Client.Update(req.Ctx, sec); err != nil {
			return err
		}
	}

	if !strings.EqualFold(*cfg.LetsEncrypt, "disabled") && domain != "" {
		if err := tls.ProvisionWildcardCert(req, resp, domain); err != nil {
			return err
		}
	}

	return nil
}

// purgeRecordsIfDisabling checks if we are transitioning AcorDNS from an enabled state to a disabled state and if so calls the
// acorn DNS service to purge all records for the domain. It is expected that string variable returned by this function
// will be set as the labels.AcornDNSState annotation on the acorn-dns secret
func purgeRecordsIfDisabling(req router.Request, domain string, cfg *apiv1.Config, dnsSecret *corev1.Secret,
	token string, dnsClient dns.Client) (string, error) {
	var state string
	if domain == "" {
		state = *cfg.AcornDNS
	} else {
		// The config object includes the acorn domain in the list of ClusterDomains if we are in an enabled state.
		if slices.Contains(cfg.ClusterDomains, domain) {
			state = "enabled"
		} else {
			state = "disabled"
		}
	}

	// purge the records if the current state is disabled, but that isn't the state recorded on the dnsSecret
	// If this is the first time the handler is being called, then the domain and token will be blank and thus we
	// shouldn't call purge
	if strings.EqualFold(state, "disabled") && dnsSecret.Annotations[labels.AcornDNSState] != "disabled" {
		if domain != "" && token != "" {
			if err := dnsClient.PurgeRecords(*cfg.AcornDNSEndpoint, domain, token); err != nil {
				if dns.IsDomainAuthError(err) {
					if err := dns.ClearDNSToken(req.Ctx, req.Client, dnsSecret); err != nil {
						return "", err
					}
				}
				return "", err
			}
		}
	}
	return state, nil
}
