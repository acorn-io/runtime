package scheduling

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
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

func TestSameGenerationMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/same-generation", Calculate)
}
