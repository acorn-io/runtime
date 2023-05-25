package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"
)

// TestToRecordRequests is a table test that asserts for given Ingress values, the expected RecordRequests are returned
func TestToRecordRequests(t *testing.T) {
	// Three most basic cases
	assrt(t, ".foo.com", []string{"app.foo.com"}, []string{"127.0.0.1"}, nil, nil, []RecordRequest{{"app", RecordTypeA, []string{"127.0.0.1"}}})
	assrt(t, ".foo.com", []string{"app.foo.com"}, nil, []string{"::1"}, nil, []RecordRequest{{"app", RecordTypeAAAA, []string{"::1"}}})
	assrt(t, ".foo.com", []string{"app.foo.com"}, nil, nil, []string{"hostname.com"}, []RecordRequest{{"app", RecordTypeCname, []string{"hostname.com"}}})

	// If the domain isn't a suffix of one of the rules, then we shouldn't create any RRs
	assrt(t, ".bar.com", []string{"app.foo.com"}, []string{"127.0.0.1"}, nil, nil, nil)
	// Similar, but ONE of the rules hosts names matches, expect RRs for just that rule
	assrt(t, ".bar.com", []string{"app.foo.com", "second.bar.com"}, []string{"127.0.0.1"}, nil, nil, []RecordRequest{{"second", RecordTypeA, []string{"127.0.0.1"}}})

	// IPv4 and 6
	assrt(t, ".foo.com", []string{"app.foo.com"}, []string{"127.0.0.1"}, []string{"::1"}, nil,
		[]RecordRequest{{"app", RecordTypeA, []string{"127.0.0.1"}}, {"app", RecordTypeAAAA, []string{"::1"}}})

	// If we have a hostname and IPv4, we only expect CNAMES
	assrt(t, ".foo.com", []string{"app.foo.com"}, []string{"127.0.0.1"}, nil, []string{"hostname.com"}, []RecordRequest{{"app", RecordTypeCname, []string{"hostname.com"}}})

	// If the only hostname is "localhost", we can't actually CNAME to localhost, so expect an A record for 127.0.0.1
	assrt(t, ".foo.com", []string{"app.foo.com"}, nil, nil, []string{"localhost"}, []RecordRequest{{"app", RecordTypeA, []string{"127.0.0.1"}}})
}

func assrt(t *testing.T, domain string, specRulesHosts, statusIPv4s, statusIPv6s, statusHosts []string, expectedRRs []RecordRequest) {
	t.Helper()

	ingress := ing(specRulesHosts, statusIPv4s, statusIPv6s, statusHosts)
	recordReqs, _ := ToRecordRequestsAndHash(domain, ingress)
	assert.Equal(t, expectedRRs, recordReqs)
}

func ing(specRulesHosts, statusIPv4s, statusIPv6s, statusHosts []string) *v1.Ingress {
	var rules []v1.IngressRule
	for _, h := range specRulesHosts {
		rules = append(rules, v1.IngressRule{Host: h})
	}

	stat := v1.IngressStatus{
		LoadBalancer: v1.IngressLoadBalancerStatus{
			Ingress: []v1.IngressLoadBalancerIngress{},
		},
	}

	for _, ip := range statusIPv4s {
		stat.LoadBalancer.Ingress = append(stat.LoadBalancer.Ingress, v1.IngressLoadBalancerIngress{IP: ip})
	}

	for _, ip := range statusIPv6s {
		stat.LoadBalancer.Ingress = append(stat.LoadBalancer.Ingress, v1.IngressLoadBalancerIngress{IP: ip})
	}

	for _, host := range statusHosts {
		stat.LoadBalancer.Ingress = append(stat.LoadBalancer.Ingress, v1.IngressLoadBalancerIngress{Hostname: host})
	}

	return &v1.Ingress{
		Spec: v1.IngressSpec{
			Rules: rules,
		},
		Status: stat,
	}
}
