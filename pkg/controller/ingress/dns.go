package ingress

import (
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/dns"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/strings/slices"
)

// RequireLBs is middleware that ensures the ingress object has at least one ingress.Status.LoadBalancer.Ingress set
// The handler in this package only operates on ingresses with that condition.
func RequireLBs(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		ingress := req.Object.(*netv1.Ingress)

		if len(ingress.Status.LoadBalancer.Ingress) == 0 {
			return nil
		}
		return h.Handle(req, resp)
	})
}

type handler struct {
	dnsClient dns.Client
}

func NewDNSHandler() router.Handler {
	s := &handler{
		dns.NewClient(),
	}
	return s
}

// Handle calls the AcornDNS service to create records for ingresses if the acorn-dns feature is enabled
func (h *handler) Handle(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{}
	if err := req.Client.Get(req.Ctx, router.Key(system.Namespace, system.DNSSecretName), secret); err != nil {
		if apierrors.IsNotFound(err) {
			// DNS Secret doesn't exist. Nothing to do
			return nil
		}
		return err
	}
	domain := string(secret.Data["domain"])
	token := string(secret.Data["token"])
	if domain == "" || token == "" {
		logrus.Infof("DNS secret missing domain (%v) or token. Won't requeset AcornDNS FQDN.", domain)
		return nil
	}

	ingress := req.Object.(*netv1.Ingress)
	var hash string
	var requests []dns.RecordRequest
	if slices.Contains(cfg.ClusterDomains, domain) {
		requests, hash = dns.ToRecordRequestsAndHash(domain, ingress)
		if len(requests) == 0 {
			return nil
		}

		if hash == ingress.Annotations[labels.AcornDNSHash] {
			// If the hashes are the same, we've already made all the appropriate DNS entries for this ingress.
			return nil
		}

		if err := h.dnsClient.CreateRecords(*cfg.AcornDNSEndpoint, domain, token, requests); err != nil {
			if dns.IsDomainAuthError(err) {
				if err := dns.ClearDNSToken(req.Ctx, req.Client, secret); err != nil {
					return err
				}
			}
			return err
		}
	}

	if hash != ingress.Annotations[labels.AcornDNSHash] {
		ingress.Annotations[labels.AcornDNSHash] = hash
		err = req.Client.Update(req.Ctx, ingress)
		if err != nil {
			return err
		}
		resp.Objects(ingress)
	}

	return nil
}
