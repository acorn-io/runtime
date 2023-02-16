package scheduling

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestContainerTolerations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/tolerations/container", Calculate)
}

func TestJobTolerations(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/tolerations/job", Calculate)
}
