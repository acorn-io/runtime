package appdefinition

import (
	"os"
	"testing"

	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
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
