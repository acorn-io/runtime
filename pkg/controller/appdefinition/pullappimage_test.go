package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	"github.com/ibuildthecloud/herd/pkg/scheme"
)

func TestPullAppImageCreateJob(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullappimage/createjob", PullAppImage("appimageinit"))
}

func TestPullAppImageRecordJob(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullappimage/recordjob", PullAppImage("appimageinit"))
}

func TestPullAppImageRecordJobError(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/pullappimage/recordjoberror", PullAppImage("appimageinit"))
}
