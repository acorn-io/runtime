package dns

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ToRecordRequestsAndHash creates DNS records based on the ingress and domain supplied. It also returns a hash of those
// records, suitable for using overtime to determine if the ingress's records need to change.
func ToRecordRequestsAndHash(domain string, ingress *v1.Ingress) ([]RecordRequest, string) {
	var ipv4s, ipv6s, lbHosts, recordValues []string

	for _, i := range ingress.Status.LoadBalancer.Ingress {
		if i.IP != "" {
			if strings.Contains(i.IP, ":") {
				ipv6s = append(ipv6s, i.IP)
			} else {
				ipv4s = append(ipv4s, i.IP)
			}
		}
		if i.Hostname != "" {
			lbHosts = append(lbHosts, i.Hostname)
		}
	}

	var hosts []string
	for _, rule := range ingress.Spec.Rules {
		if strings.HasSuffix(rule.Host, domain) {
			hosts = append(hosts, strings.TrimSuffix(rule.Host, domain))
		}
	}

	var requests []RecordRequest
	if len(lbHosts) > 0 {
		var recordType string
		if len(lbHosts) == 1 && lbHosts[0] == "localhost" {
			recordValues = []string{"127.0.0.1"}
			recordType = "A"
		} else {
			recordValues = lbHosts
			recordType = "CNAME"
		}

		for _, host := range hosts {
			requests = append(requests, RecordRequest{
				Name:   host,
				Type:   RecordType(recordType),
				Values: recordValues,
			})
		}
	} else {
		if len(ipv4s) > 0 {
			recordValues = append(recordValues, ipv4s...)
			for _, host := range hosts {
				requests = append(requests, rr(host, "A", ipv4s))
			}
		}
		if len(ipv6s) > 0 {
			recordValues = append(recordValues, ipv6s...)
			for _, host := range hosts {
				requests = append(requests, rr(host, "AAAA", ipv6s))
			}
		}
	}

	hash := generateHash(domain, hosts, recordValues)
	return requests, hash
}

func rr(host, rType string, recordVals []string) RecordRequest {
	return RecordRequest{
		Name:   host,
		Type:   RecordType(rType),
		Values: recordVals,
	}
}

func generateHash(domain string, hosts, ips []string) string {
	toHash := domain

	sort.Slice(hosts, func(i, j int) bool {
		return i < j
	})
	sort.Slice(ips, func(i, j int) bool {
		return i < j
	})
	for _, h := range hosts {
		toHash += h + ","
	}
	for _, i := range ips {
		toHash += i + ","
	}

	dig := sha1.New()
	dig.Write([]byte(toHash))
	return hex.EncodeToString(dig.Sum(nil))
}

// ClearDNSToken deletes the token used to authenticate to the AcornDNS service.
func ClearDNSToken(ctx context.Context, client kclient.Client, dnsSecret *corev1.Secret) error {
	logrus.Infof("Clearing token for domain %s", dnsSecret.Data["domain"])
	delete(dnsSecret.Data, "token")
	return client.Update(ctx, dnsSecret)
}
