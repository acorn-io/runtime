package scheduling

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestContainerWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/container", Calculate)
}

func TestDifferentWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/different-workloadclass", Calculate)
}

func TestJobWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/job", Calculate)
}

func TestOverwriteAcornfileWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/overwrite-acornfile-workloadclass", Calculate)
}

func TestWithAcornfileWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/with-acornfile-workloadclass", Calculate)
}

func TestTwoContainersWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/two-containers", Calculate)
}

func TestAllSetWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/all-set", Calculate)
}

func TestAllSetOverwriteWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/all-set-overwrite-workloadclass", Calculate)
}

func TestSameGenerationWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/same-generation", Calculate)
}

func TestTwoCWCDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/workloadclass/two-cwc-defaults-should-error")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}

func TestTwoPWCDefaultsShouldError(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/workloadclass/two-pwc-defaults-should-error")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := harness.Invoke(t, input, router.HandlerFunc(Calculate))
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, resp.NoPrune, "NoPrune should be true when error occurs")
}
