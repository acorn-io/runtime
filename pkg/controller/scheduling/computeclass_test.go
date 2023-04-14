package scheduling

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestContainerComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/container", Calculate)
}

func TestDifferentComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/different-computeclass", Calculate)
}

func TestJobComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/job", Calculate)
}

func TestOverwriteAcornfileComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/overwrite-acornfile-computeclass", Calculate)
}

func TestWithAcornfileComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/with-acornfile-computeclass", Calculate)
}

func TestTwoContainersComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/two-containers", Calculate)
}

func TestAllSetComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set", Calculate)
}

func TestAllSetOverwriteComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set-overwrite-computeclass", Calculate)
}

func TestSameGenerationComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/same-generation", Calculate)
}

func TestSameDigestGenerationComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/same-digest-generation", Calculate)
}

func TestDifferentDigestGenerationComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/different-digest-generation", Calculate)
}

func TestTwoCCCDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/computeclass/two-ccc-defaults-should-error")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}

func TestTwoPCCDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/computeclass/two-pcc-defaults-should-error")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}
