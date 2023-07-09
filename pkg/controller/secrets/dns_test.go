package secrets

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestCreateIngressAcornDNSEnabled(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/dns/create-ingress-acorndns-enabled", HandleDNSSecret)
}

func TestCreateIngressClusterDomain(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/dns/create-ingress-cluster-domain", HandleDNSSecret)
}
