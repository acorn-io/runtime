package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestDepends(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/depends", DeploySpec)
}

func TestDependsReadyReplicaSet(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/depends-ready", DeploySpec)
}
