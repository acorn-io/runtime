package appdefinition

import (
	"testing"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/z"
	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAcornLabels(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/labels", DeploySpec)
}

func TestAcornBasic(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/acorn/basic", DeploySpec)
}

func TestComputeMem(t *testing.T) {
	h := tester.Harness{
		Scheme: scheme.Scheme,
	}

	resp, err := h.Invoke(t, &v1.AppInstance{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "app-namespace",
		},
		Spec: v1.AppInstanceSpec{
			Memory: map[string]*int64{
				"":             z.Pointer(int64(1)),
				"byname":       z.Pointer(int64(2)),
				"byname.child": z.Pointer(int64(3)),
			},
			ComputeClasses: map[string]string{
				"":             "defaultValue",
				"byname":       "byNameValue",
				"byname.child": "byNameChildValue",
			},
		},
		Status: v1.AppInstanceStatus{
			Namespace: "app-created-namespace",
			AppImage: v1.AppImage{
				ID: "foo",
			},
			AppSpec: v1.AppSpec{
				Acorns: map[string]v1.Acorn{
					"byname": {
						Image: "foo",
					},
					"other": {
						Image: "foo",
					},
				},
			},
		},
	}, router.HandlerFunc(DeploySpec))

	require.NoError(t, err)

	autogold.ExpectFile(t, h.SanitizedYAML(t, resp.Collected))
}
