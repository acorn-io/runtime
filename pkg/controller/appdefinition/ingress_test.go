package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/basic", DeploySpec)
}

func TestIngressClusterDomainWithPort(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/clusterdomainport", DeploySpec)
}
