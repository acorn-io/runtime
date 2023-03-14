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

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		cidr        string
		errExpected bool
	}{
		{
			cidr:        "0.0.0.0/0",
			errExpected: false,
		},
		{
			cidr:        "1.3.4.5/32",
			errExpected: false,
		},
		{
			cidr:        "000.111.032.45/03", // weird but technically valid
			errExpected: false,
		},
		{
			cidr:        "255.255.0.13/10",
			errExpected: false,
		},
		{
			cidr:        "256.0.0.0/24",
			errExpected: true,
		},
		{
			cidr:        "1a1.0.0.0/31",
			errExpected: true,
		},
		{
			cidr:        "11.11.11.11/33",
			errExpected: true,
		},
		{
			cidr:        "this is not a cidr",
			errExpected: true,
		},
		{
			cidr:        "1.2.3/4",
			errExpected: true,
		},
		{
			cidr:        "1.1.0.0/24/2",
			errExpected: true,
		},
		{
			cidr:        "10.10.0.0/24a",
			errExpected: true,
		},
		{
			cidr:        "1.2.3.4./16",
			errExpected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.cidr, func(t *testing.T) {
			err := validateCIDR(test.cidr)
			if err != nil && !test.errExpected {
				t.Fatal(err)
			}
		})
	}
}
