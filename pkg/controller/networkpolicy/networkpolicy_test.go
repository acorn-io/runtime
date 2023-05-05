package networkpolicy

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestNetworkPolicyForApp(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/appinstance", ForApp)
}

func TestNetworkPolicyForIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingress", ForIngress)
}

func TestNetworkPolicyForIngressExternalName(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/externalname", ForIngress)
}

func TestNetworkPolicyForService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/service", ForService)
}

func TestNetworkPolicyForBuilder(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/builder", ForBuilder)
}
