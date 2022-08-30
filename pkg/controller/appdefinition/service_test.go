package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestService(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/basic", DeploySpec)
}

func TestAlias(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/service/alias", DeploySpec)
}
