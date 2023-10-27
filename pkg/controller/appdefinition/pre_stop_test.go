package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestPreStop(t *testing.T) {
	acornSleepBinary = []byte("acorn-sleep")
	t.Cleanup(func() {
		acornSleepBinary = nil
	})

	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/basic", DeploySpec)
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/stateful", DeploySpec)
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/job", DeploySpec)
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/dev", DeploySpec)
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/no-ports", DeploySpec)
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec/pre-stop/ports-only-sidecar", DeploySpec)
}
