package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	"github.com/ibuildthecloud/herd/pkg/scheme"
)

func TestAssignNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assignnamespace", AssignNamespace)
}
