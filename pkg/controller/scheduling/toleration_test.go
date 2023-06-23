package scheduling

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestContainerTolerations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/tolerations/container", Calculate)
}

func TestJobTolerations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/tolerations/job", Calculate)
}
