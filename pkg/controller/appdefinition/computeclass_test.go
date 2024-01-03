package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestContainerComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/container", DeploySpec)
}

func TestDifferentComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/different-computeclass", DeploySpec)
}

func TestJobComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/job", DeploySpec)
}

func TestOverwriteAcornfileComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/overwrite-acornfile-computeclass", DeploySpec)
}

func TestWithAcornfileComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/with-acornfile-computeclass", DeploySpec)
}

func TestTwoContainersComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/two-containers", DeploySpec)
}

func TestAllSetComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set", DeploySpec)
}

func TestAllSetOverwriteComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/all-set-overwrite-computeclass", DeploySpec)
}

func TestGenericResourcesComputeClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/computeclass/generic-resources", DeploySpec)
}
