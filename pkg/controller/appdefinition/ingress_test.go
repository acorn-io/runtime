package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/basic", DeploySpec)
}
func TestIngressPrefix(t *testing.T) {
	path := "testdata/ingress/prefix/prefix-1"
	var resp *tester.Response
	var resp2 *tester.Response

	t.Run(path, func(t *testing.T) {
		harness, input, err := tester.FromDir(scheme.Scheme, path)
		if err != nil {
			t.Fatal(err)
		}
		resp, err = harness.Invoke(t, input, router.HandlerFunc(DeploySpec))
		if err != nil {
			t.Fatal(err)
		}
	})
	path = "testdata/ingress/prefix/prefix-2"
	t.Run(path, func(t *testing.T) {
		harness, input, err := tester.FromDir(scheme.Scheme, path)
		if err != nil {
			t.Fatal(err)
		}
		resp2, err = harness.Invoke(t, input, router.HandlerFunc(DeploySpec))
		if err != nil {
			t.Fatal(err)
		}
	})
	assert.Equal(t, len(resp.Collected), len(resp2.Collected))
	var index1 int
	var index2 int
	for index, yaml := range resp.Collected {
		if _, ok := yaml.(*v1.Ingress); ok {
			index1 = index
		}
	}
	for index, yaml := range resp2.Collected {
		if _, ok := yaml.(*v1.Ingress); ok {
			index2 = index
		}
	}
	assert.True(t, cmp.Equal(resp.Collected[index1], resp2.Collected[index2]))
}

func TestIngressPrefix1(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-1", DeploySpec)
}

func TestIngressPrefix2(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-2", DeploySpec)
}

func TestIngressPrefix1Namespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-1-namespace", AddNamespace)
}

func TestIngressPrefix2Namespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-2-namespace", AddNamespace)
}

func TestIngressClusterDomainWithPort(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/clusterdomainport", DeploySpec)
}

func TestIngressLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/labels", DeploySpec)
}

func TestIngressLabelsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/labels-namespace", AddNamespace)
}

func TestLetsEncrypt(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/letsencrypt", DeploySpec)
}
