package defaults

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestCalculateRegionDefault(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/region/default", Calculate)
}

func TestCalculateRegionDefaultProjectStatus(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/region/project-default-status", Calculate)
}

func TestCalculateRegionDefaultOnSpec(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/region/region-on-spec", Calculate)
}
