package networkpolicy

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestNetworkPolicyForApp(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/appinstance", PolicyForApp)
}

func TestNetworkPolicyForIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingress", PolicyForIngress)
}

func TestNetworkPolicyForIngressExternalName(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/externalname", PolicyForIngress)
}

func TestNetworkPolicyForService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/service", PolicyForService)
}

func TestNetworkPolicyForBuilder(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/builder", PolicyForBuilder)
}
