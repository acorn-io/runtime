package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestDepends(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/depends", func(req router.Request, resp router.Response) error {
		return CheckDependencies(router.HandlerFunc(DeploySpec)).Handle(req, resp)
	})
}
