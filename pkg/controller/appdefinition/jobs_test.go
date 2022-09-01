package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestJobs(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/basic", DeploySpec)
}

func TestJobsLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/labels", DeploySpec)
}

func TestCronJobs(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/cronjob", DeploySpec)
}
