package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestAssignNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assignnamespace", AssignNamespace)
}
