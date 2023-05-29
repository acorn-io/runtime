package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestDepends(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/depends", DeploySpec)
}

func TestDependsReadyReplicaSet(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/depends-ready", DeploySpec)
}
