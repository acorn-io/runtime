package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestContainerMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/container", DeploySpec)
}

func TestSidecarMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/sidecar", DeploySpec)
}

func TestJobMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/job", DeploySpec)
}

func TestOverwriteAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/overwrite-acornfile-memory", DeploySpec)
}

func TestWithAcornfileMemory(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/with-acornfile-memory", DeploySpec)
}

func TestTwoContainers(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/two-containers", DeploySpec)
}

func TestAllSet(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/all-set", DeploySpec)
}

func TestAllSetOverwrite(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/memory/all-set-overwrite", DeploySpec)
}
