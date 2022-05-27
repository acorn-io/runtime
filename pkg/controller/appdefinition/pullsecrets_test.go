package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestPullSecrets(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullsecrets/default", DeploySpec)
}

func TestPullSecretsCustom(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullsecrets/custom", DeploySpec)
}
