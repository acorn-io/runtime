package appdefinition

import (
	"os"
	"testing"

	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/acorn-io/runtime/pkg/controller/namespace"
	"github.com/acorn-io/runtime/pkg/scheme"
)

func TestJobs(t *testing.T) {
	dirs, err := os.ReadDir("testdata/job")
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range dirs {
		name := dir.Name()
		if dir.IsDir() && name != "labels-namespace" {
			t.Run(name, func(t *testing.T) {
				tester.DefaultTest(t, scheme.Scheme, "testdata/job/"+name, DeploySpec)
			})
		}
	}
}

func TestJobsLabelsNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/job/labels-namespace", namespace.AddNamespace)
}
