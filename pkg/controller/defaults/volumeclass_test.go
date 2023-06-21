package defaults

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/assert"
)

func TestCalculateSameGeneration(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-defaults-same-gen", Calculate)
}

func TestFillVolumeClassDefaults(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-fill-defaults", Calculate)
}

func TestVolumeClassFillSizeDefaults(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-fill-size", Calculate)
}

func TestProjectVolumeClassDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-fill-project-default", Calculate)
}

func TestClusterVolumeClassDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-fill-cluster-default", Calculate)
}

func TestClusterProjectWithSameName(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/cluster-and-project-class-same-name", Calculate)
}

func TestFillVolumeClassDefaultsWithVolumeBinding(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/volumeclass/volume-class-fill-defaults-with-bind", Calculate)
}

func TestTwoClusterDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/volumeclass/volume-class-two-cluster-defaults")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}

func TestTwoProjectDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/volumeclass/volume-class-two-project-defaults")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}
