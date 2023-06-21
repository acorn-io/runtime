package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestLink(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/link", DeploySpec)
}
