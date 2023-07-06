package ingress

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	mocks "github.com/acorn-io/runtime/pkg/mocks/dns"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

// TestBasicDNS is a simple test that asserts the basic logic of calling the DNS service to create an FQDN for the
// acorn-ingress Ingress.
//
// To turn on AcornDNS integration, existing.yaml supplies the acorn-config configMap and acorn-dns secret. The configMap
// sets acornDNS to "enabled". The secret has the expected domain and token fields.
//
// input.yaml supplies an ingress that reflects what acorn-ingress will look like where its status.LoadBalancer.ingress
// field is set (required to create an FQDN).
//
// expected.yaml has the same ingress that is in input.yaml, but with the acorn.io/dns-hash annotation set, which indicates
// the FQDN was successfully created in the DNS service.
//
// The client to the DNS service is mocked out.
func TestBasicDNS(t *testing.T) {
	h := &handler{
		dnsClient: mockDNSClient(t),
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

func mockDNSClient(t *testing.T) *mocks.MockClient {
	t.Helper()

	ctrl := gomock.NewController(t)
	dnsClient := mocks.NewMockClient(ctrl)

	// Register expected mock client calls
	dnsClient.EXPECT().CreateRecords(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	return dnsClient
}
