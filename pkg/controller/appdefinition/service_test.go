package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/basic", DeploySpec)
}

func TestAlias(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/alias", DeploySpec)
}
