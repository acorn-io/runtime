package appdefinition

import (
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"testing"
)

func TestNetworkPolicy(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/default", NetworkPolicy)
}

func TestNetworkPolicyWithIngressNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingressnamespace", NetworkPolicy)
}

func TestNetworkPolicyWithPodCIDR(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/podcidr", NetworkPolicy)
}
