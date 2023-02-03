package appdefinition

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/controller/namespace"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAssignNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assignnamespace", AssignNamespace)
}

func TestAssignTargetNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/assigntargetnamespace", AssignNamespace)
}

func TestLabelsAnnotationsBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/propagation_basic", namespace.AddNamespace)
}

func TestLabelsAnnotationsNoConfigset(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/propagation_noconfig", namespace.AddNamespace)
}

func TestHandler_AddAcornProjectLabel(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/addacornprojectlabel")
	if err != nil {
		t.Fatal(err)
	}

	req := tester.NewRequest(t, harness.Scheme, input, harness.Existing...)

	if err = AddAcornProjectLabel(req, nil); err != nil {
		t.Fatal(err)
	}
	var projectNamespace v1.Namespace
	if err = req.Client.Get(req.Ctx, kclient.ObjectKey{
		Name: input.GetNamespace(),
	}, &projectNamespace); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "true", projectNamespace.Labels[labels.AcornProject])
}

func TestHandler_AddAcornProjectLabelAlreadyExists(t *testing.T) {
	harness, input, err := tester.FromDir(scheme.Scheme, "testdata/addacornprojectlabelalreadyexists")
	if err != nil {
		t.Fatal(err)
	}

	req := tester.NewRequest(t, harness.Scheme, input, harness.Existing...)

	if err = AddAcornProjectLabel(req, nil); err != nil {
		t.Fatal(err)
	}
	var projectNamespace v1.Namespace
	if err = req.Client.Get(req.Ctx, kclient.ObjectKey{
		Name: input.GetNamespace(),
	}, &projectNamespace); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "true", projectNamespace.Labels[labels.AcornProject])
}
