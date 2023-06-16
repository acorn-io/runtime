package service

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"
)

func TestIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/basic", RenderServices)
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
		resp, err = harness.Invoke(t, input, router.HandlerFunc(RenderServices))
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
		resp2, err = harness.Invoke(t, input, router.HandlerFunc(RenderServices))
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
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-1", RenderServices)
}

func TestIngressPrefix2(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-2", RenderServices)
}

func TestIngressPrefix1Namespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-1-namespace", namespace.AddNamespace)
}

func TestIngressPrefix2Namespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/prefix/prefix-2-namespace", namespace.AddNamespace)
}

func TestIngressClusterDomainWithPort(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/clusterdomainport", RenderServices)
}

func TestIngressLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/labels", RenderServices)
}

func TestIngressLabelsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/labels-namespace", namespace.AddNamespace)
}

func TestLetsEncrypt(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/letsencrypt", RenderServices)
}

func TestService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/basic", RenderServices)
}

func TestServiceOverlay(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/tcp-http-overlap", RenderServices)
}

func TestBindNoProtocol(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/bind-no-protocol", RenderServices)
}

func TestRouter(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/router", RenderServices)
}

func TestSecret(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/secret", RenderServices)
}

func TestCertManager(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress/cert-manager", RenderServices)
}
