package defaults

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestContainerMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/container", Calculate)
}

func TestSidecarMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/sidecar", Calculate)
}

func TestJobMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/job", Calculate)
}

func TestOverwriteAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/overwrite-acornfile-memory", Calculate)
}

func TestWithAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/with-acornfile-memory", Calculate)
}

func TestTwoContainers(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/two-containers", Calculate)
}

func TestAllSet(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/all-set", Calculate)
}

func TestAllSetOverwrite(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/all-set-overwrite", Calculate)
}

func TestMemorySameGeneration(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/same-generation", Calculate)
}

func TestTwoCCCDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/memory/two-ccc-defaults-should-error")
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
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/memory/two-pcc-defaults-should-error")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}
