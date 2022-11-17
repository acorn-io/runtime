package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestContainer(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/container", DeploySpec)
}

func TestMultipleContainers(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/multiplecontainers", DeploySpec)
}

func TestDifferentPermissions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/differentpermissions", DeploySpec)
}

func TestJob(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/job", DeploySpec)
}

func TestMultipleJobs(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/multiplejobs", DeploySpec)
}

func TestBoth(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/both", DeploySpec)
}

func TestBothWithNoPermissions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/permissions/bothwithnopermissions", DeploySpec)
}
