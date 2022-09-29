package dns

import (
	"strings"
	"time"

	"github.com/acorn-io/acorn/pkg/dns"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/sirupsen/logrus"
)

const (
	AuthorizationHeader = "Authorization"
	ContentTypeHeader   = "Content-Type"
	ContentTypeJSON     = "application/json"
	txtPathPattern      = "%s/domain/%s/txt"
)

/*
 * DNS01 Challenge Solver (Lego Interface)
 */

type DNSProvider struct {
	client DNSClient
}

func NewDNSProvider(endpoint, domain, token string) *DNSProvider {
	return &DNSProvider{
		client: NewDNSClient(endpoint, domain, token),
	}
}

func (d *DNSProvider) Present(domain, token, keyAuth string) error {
	fqdn, value := dns01.GetRecord(domain, keyAuth)
	return d.client.SetTXTRecord(fqdn, value)
}

func (d *DNSProvider) CleanUp(domain, token, keyAuth string) error {
	return d.client.DeleteDNSRecord(domain)
}

func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return 3 * time.Minute, 1 * time.Minute
}

/*
 * AcornDNS Helper
 */

type DNSClient struct {
	dns      dns.Client
	domain   string
	token    string
	endpoint string
}

func NewDNSClient(endpoint, domain, token string) DNSClient {
	return DNSClient{
		dns:      dns.NewClient(),
		domain:   domain,
		token:    token,
		endpoint: endpoint,
	}
}

func (d *DNSClient) SetTXTRecord(domain, text string) error {

	prefix := strings.TrimSuffix(strings.TrimSuffix(domain, "."), d.domain)

	var requests []dns.RecordRequest
	requests = append(requests, dns.RecordRequest{
		Name:   prefix,
		Type:   dns.RecordTypeTxt,
		Values: []string{text},
	})

	logrus.Debugf("Setting TXT record %s - %s for domain %s", prefix, text, d.domain)

	return d.dns.CreateRecords(d.endpoint, d.domain, d.token, requests)
}

func (d *DNSClient) DeleteDNSRecord(domain string) error {
	prefix := strings.TrimSuffix(strings.TrimSuffix(domain, "."), d.domain)
	d.dns.DeleteRecord(d.endpoint, d.domain, prefix, d.token)
	return nil
}
