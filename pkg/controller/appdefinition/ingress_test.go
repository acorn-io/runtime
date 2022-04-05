package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	"github.com/ibuildthecloud/herd/pkg/scheme"
)

func TestIngress(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/ingress", DeploySpec)
}
