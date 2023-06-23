package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestAcornLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/labels", DeploySpec)
}

func TestAcornBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/basic", DeploySpec)
}
