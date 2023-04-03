package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestNetworkPolicyForApp(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/appinstance", NetworkPolicyForApp)
}

func TestNetworkPolicyForIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingress", NetworkPolicyForIngress)
}

func TestNetworkPolicyForIngressExternalName(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/externalname", NetworkPolicyForIngress)
}

func TestNetworkPolicyForService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/service", NetworkPolicyForService)
}
