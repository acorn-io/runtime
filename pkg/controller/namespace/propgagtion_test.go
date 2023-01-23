package namespace

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
)

func TestLabelsAnnotationsBasic(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/basic")
	if err != nil {
		t.Fatal(err)
	}
	req := tester.NewRequest(t, harness.Scheme, input, harness.Existing...)
	if err := LabelsAnnotations(req, nil); err != nil {
		t.Fatal(err)
	}

	// Check whether annotations and labels from project are propagated
	assert.Equal(t, "foo1", harness.Existing[1].GetAnnotations()["foo"])
	assert.Equal(t, "bar1", harness.Existing[1].GetLabels()["bar"])
}

func TestLabelsAnnotationsNoConfigset(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/noconfig")
	if err != nil {
		t.Fatal(err)
	}
	req := tester.NewRequest(t, harness.Scheme, input, harness.Existing...)
	if err := LabelsAnnotations(req, nil); err != nil {
		t.Fatal(err)
	}

	// No propagation config is set, so it shouldn't populate labels and annotations
	assert.Equal(t, "", harness.Existing[1].GetAnnotations()["foo"])
	assert.Equal(t, "", harness.Existing[1].GetAnnotations()["bar"])
}

func TestLabelsAnnotationsShouldNotPopulateToOtherNamespaces(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/other")
	if err != nil {
		t.Fatal(err)
	}
	req := tester.NewRequest(t, harness.Scheme, input, harness.Existing...)
	if err := LabelsAnnotations(req, nil); err != nil {
		t.Fatal(err)
	}

	// No propagation since the namespace is not an Acorn app namespace
	assert.Equal(t, "", harness.Existing[1].GetAnnotations()["foo"])
	assert.Equal(t, "", harness.Existing[1].GetAnnotations()["bar"])
}
