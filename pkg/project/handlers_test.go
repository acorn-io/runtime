package project

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestSetProjectSupportedRegions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/no-default", SetProjectSupportedRegions)
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/with-supported-regions", SetProjectSupportedRegions)
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/with-default-and-supported", SetProjectSupportedRegions)
}

func TestCreateNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/createnamespace/without-labels-anns", CreateNamespace)
	tester.DefaultTest(t, scheme.Scheme, "testdata/createnamespace/with-labels-anns", CreateNamespace)
}
