package ingress

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/dns"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

// TestBasicDNS is a simple test that asserts the basic logic of calling the DNS service to create an FQDN for an
// acorn created ingress.
//
// To turn on AcornDNS integration, existing.yaml supplies the acorn-config configMap and acorn-dns secret. The configMap
// sets acornDNS to "enabled". The secret has the expected domain and token fields.
//
// input.yaml supplies an ingress that has the status.LoadBalancer.ingress field set (required to create an FQDN).
//
// expected.yaml has the same ingress that is in input.yaml, but with the acorn.io/dns-hash annotation set, which indicates
// the FQDN was successfully created in the DNS service.
//
// The client to the DNS service is mocked out.
func TestBasicDNS(t *testing.T) {
	h := &handler{
		dnsClient: &mockClient{},
	}

	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/ingress")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, h)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, resp.Client.Updated, 1)
	ingress := resp.Client.Updated[0]
	assert.NotEmpty(t, ingress.GetAnnotations()["acorn.io/dns-hash"])
}

// TODO Use a mock library to create more robust mock for this. Right now, just CreateRecord has been implemented to
// simply not panic. This is enough for the handler to assume the call succeeded and move on
type mockClient struct{}

func (t *mockClient) CreateRecords(endpoint, domain, token string, records []dns.RecordRequest) error {
	return nil
}

func (t *mockClient) ReserveDomain(endpoint string) (string, string, error) {
	return "test.oss-acorn.io", "token", nil
}

func (t *mockClient) Renew(endpoint, domain, token string, renew dns.RenewRequest) (dns.RenewResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (t *mockClient) DeleteRecord(endpoint, domain, fqdn, token string) error {
	//TODO implement me
	panic("implement me")
}

func (t *mockClient) PurgeRecords(endpoint, domain, token string) error {
	return nil
}
