package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestAssignNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assignnamespace", AssignNamespace)
}

func TestAssignTargetNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assigntargetnamespace", AssignNamespace)
}

func TestLabelsAnnotationsBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/propagation_basic", namespace.AddNamespace)
}

func TestLabelsAnnotationsNoConfigset(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/propagation_noconfig", namespace.AddNamespace)
}
