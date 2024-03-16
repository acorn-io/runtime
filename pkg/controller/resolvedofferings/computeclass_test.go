package resolvedofferings

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/assert"
)

func TestContainerMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/container", Calculate)
}

func TestSidecarMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/sidecar", Calculate)
}

func TestJobMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/job", Calculate)
}

func TestOverwriteAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/overwrite-acornfile-memory", Calculate)
}

func TestWithAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/with-acornfile-memory", Calculate)
}

func TestTwoContainers(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/two-containers", Calculate)
}

func TestAllSet(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set", Calculate)
}

func TestAllSetOverwrite(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set-overwrite", Calculate)
}

func TestMemorySameGeneration(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/same-generation", Calculate)
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

func TestTwoCCCDefaultsDifferentRegions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/two-ccc-defaults-different-regions", Calculate)
}

func TestTwoPCCDefaultsDifferentRegions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/two-pcc-defaults-different-regions", Calculate)
}

func TestComputeClassDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/compute-class-default", Calculate)
}

func TestAcornfileOverrideComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/acornfile-override-compute-class", Calculate)
}

func TestUserOverrideComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/user-override-compute-class", Calculate)
}

func TestWithAcornfileMemoryAndSpecOverride(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/with-acornfile-memory-and-spec-override", Calculate)
}

func TestRemovedContainer(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/removed-container", Calculate)
}
