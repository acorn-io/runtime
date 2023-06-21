package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestRouter(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/router", DeploySpec)
}
