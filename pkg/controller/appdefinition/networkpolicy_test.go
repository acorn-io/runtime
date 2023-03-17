package appdefinition

import (
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"testing"
)

func TestNetworkPolicyForApp(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/appinstance", NetworkPolicyForApp)
}

func TestNetworkPolicyForIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingress", NetworkPolicyForIngress)
}

func TestNetworkPolicyForService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/service", NetworkPolicyForService)
}
