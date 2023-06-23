package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestPullSecrets(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullsecrets/default", DeploySpec)
}

func TestPullSecretsCustom(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullsecrets/custom", DeploySpec)
}
