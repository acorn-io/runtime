package dns

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/networking/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ToRecordRequestsAndHash creates DNS records based on the ingress and domain supplied. It also returns a hash of those
// records, suitable for using overtime to determine if the ingress's records need to change.
func ToRecordRequestsAndHash(domain string, ingress *v1.Ingress) ([]RecordRequest, string) {
	var ips, lbHosts, recordValues []string

	for _, i := range ingress.Status.LoadBalancer.Ingress {
		if i.IP != "" {
			ips = append(ips, i.IP)
		}
		if i.Hostname != "" {
			lbHosts = append(lbHosts, i.Hostname)
		}
	}
	var recordType string
	if len(ips) > 0 && len(lbHosts) > 0 {
		logrus.Warnf("Cannot create DNS for ingress %v because it has both IPs and hostnames", ingress.Name)
		return nil, ""
	} else if len(ips) > 0 {
		recordType = "A"
		recordValues = ips
	} else if len(lbHosts) > 0 {
		if len(lbHosts) == 1 && lbHosts[0] == "localhost" {
			recordType = "A"
			recordValues = []string{"127.0.0.1"}
		} else {
			recordType = "CNAME"
			recordValues = lbHosts
		}
	} else {
		return nil, ""
	}

	var hosts []string
	for _, rule := range ingress.Spec.Rules {
		if strings.HasSuffix(rule.Host, domain) {
			hosts = append(hosts, strings.TrimSuffix(rule.Host, domain))
		}
	}

	var requests []RecordRequest
	for _, host := range hosts {
		requests = append(requests, RecordRequest{
			Name:   host,
			Type:   RecordType(recordType),
			Values: recordValues,
		})
	}

	hash := generateHash(hosts, recordValues, recordType)
	return requests, hash
}

func generateHash(hosts, ips []string, recordType string) string {
	var toHash string
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
	toHash += recordType

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
