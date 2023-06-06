package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestAcornLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/labels", DeploySpec)
}

func TestAcornBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/basic", DeploySpec)
}
