package appdefinition

import (
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"testing"
)

func TestNetworkPolicy(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/default", NetworkPolicy)
}

func TestNetworkPolicyWithIngressNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/ingressnamespace", func(req router.Request, resp router.Response) error {
		conf, err := config.Get(req.Ctx, req.Client)
		if err != nil {
			return err
		}

		conf.IngressControllerNamespace = toStringPointer("nginx")
		err = config.Set(req.Ctx, req.Client, conf)
		if err != nil {
			return err
		}

		return NetworkPolicy(req, resp)
	})
}

func TestNetworkPolicyWithNodeCIDR(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/networkpolicy/nodecidr", func(req router.Request, resp router.Response) error {
		conf, err := config.Get(req.Ctx, req.Client)
		if err != nil {
			return err
		}

		conf.NodeCIDR = toStringPointer("10.2.0.1/24")
		err = config.Set(req.Ctx, req.Client, conf)
		if err != nil {
			return err
		}

		return NetworkPolicy(req, resp)
	})
}

func toStringPointer(input string) *string {
	return &input
}
