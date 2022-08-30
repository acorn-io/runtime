package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestAcorn(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/ports", DeploySpec)
}

func TestAcornLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/labels", DeploySpec)
}
