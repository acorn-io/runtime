package quota

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/basic", EnsureQuotaRequest)
}

func TestNotEnforced(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/not-enforced", EnsureQuotaRequest)
}

func TestDefaultStatusVolumeSize(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/status-default-volume-size", EnsureQuotaRequest)
}

func TestImplicitPVBind(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/implicit-pv-bind", EnsureQuotaRequest)
}

func TestOverProvisioned(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/over-provisioned", EnsureQuotaRequest)
}
