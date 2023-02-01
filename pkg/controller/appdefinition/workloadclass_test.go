package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestContainerWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/container", DeploySpec)
}

func TestDifferentWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/different-workloadclass", DeploySpec)
}

func TestJobWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/job", DeploySpec)
}

func TestOverwriteAcornfileWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/overwrite-acornfile-workloadclass", DeploySpec)
}

func TestWithAcornfileWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/with-acornfile-workloadclass", DeploySpec)
}

func TestTwoContainersWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/two-containers", DeploySpec)
}

func TestAllSetWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/all-set", DeploySpec)
}

func TestAllSetOverwriteWorkloadClass(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/workloadclass/all-set-overwrite-workloadclass", DeploySpec)
}
