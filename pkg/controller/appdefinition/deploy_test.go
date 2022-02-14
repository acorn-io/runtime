package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	"github.com/ibuildthecloud/herd/pkg/scheme"
)

func TestDeploySpec(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/deployspec", DeploySpec)
}
