package defaults

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestCalculateDefaultsSameGeneration(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-defaults-same-gen", CalculateDefaults)
}

func TestFillVolumeClassDefaults(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-fill-defaults", CalculateDefaults)
}

func TestVolumeClassFillSizeDefaults(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-fill-size", CalculateDefaults)
}

func TestProjectVolumeClassDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-fill-project-default", CalculateDefaults)
}

func TestClusterVolumeClassDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-fill-cluster-default", CalculateDefaults)
}

func TestClusterProjectWithSameName(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/cluster-and-project-class-same-name", CalculateDefaults)
}

func TestFillVolumeClassDefaultsWithVolumeBinding(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/defaults/volume-class-fill-defaults-with-bind", CalculateDefaults)
}

func TestTwoClusterDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/defaults/volume-class-two-cluster-defaults")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(CalculateDefaults))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}

func TestTwoProjectDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/defaults/volume-class-two-project-defaults")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(CalculateDefaults))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}

func TestCheckStatus(t *testing.T) {
	appInstance := new(v1.AppInstance)
	req := tester.NewRequest(t, scheme.Scheme, appInstance)
	resp := &tester.Response{Client: req.Client.(*tester.Client)}
	called := false

	handler := router.HandlerFunc(func(router.Request, router.Response) error {
		called = true
		return nil
	})

	if err := appdefinition.CheckStatus(handler).Handle(req, resp); err != nil {
		t.Fatal(err)
	}
	assert.False(t, called, "router handler call unexpected")

	condition.Setter(appInstance, resp, v1.AppInstanceConditionDefaults).Success()
	if err := appdefinition.CheckStatus(handler).Handle(req, resp); err != nil {
		t.Fatal(err)
	}

	assert.True(t, called, "router handler call expected")
}
