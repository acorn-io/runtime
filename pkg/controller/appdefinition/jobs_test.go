package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestJobs(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/basic", DeploySpec)
}

func TestJobsLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/labels", DeploySpec)
}

func TestJobsLabelsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/labels-namespace", namespace.AddNamespace)
}

func TestCronJobs(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/cronjob", DeploySpec)
}

func TestDeleteJob(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/delete-job", DeploySpec)
}
