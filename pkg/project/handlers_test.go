package project

import (
	"context"
	"testing"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/router/tester"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestSetProjectSupportedRegions(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/no-default", SetSupportedRegions)
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/with-supported-regions", SetSupportedRegions)
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/with-default-and-supported", SetSupportedRegions)
	tester.DefaultTest(t, scheme.Scheme, "testdata/setsupportedregions/all-supported-regions-with-default", SetSupportedRegions)
}

func TestCreateNamespace(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/createnamespace/without-labels-anns", CreateNamespace)
	tester.DefaultTest(t, scheme.Scheme, "testdata/createnamespace/with-labels-anns", CreateNamespace)
}

func TestEnsureAllAppsRemovedNoParents(t *testing.T) {
	input := &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-project",
		},
	}
	existing := []kclient.Object{
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-project",
			},
		},
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-other-app",
				Namespace: "my-project",
			},
		},
	}

	c := &deleteClient{
		Client: &tester.Client{
			Objects:   append(existing, input.DeepCopyObject().(kclient.Object)),
			SchemeObj: scheme.Scheme,
		},
	}
	req := router.Request{
		Client:      c,
		Object:      input,
		Ctx:         context.Background(),
		GVK:         v1.SchemeGroupVersion.WithKind("ProjectInstance"),
		Namespace:   input.GetNamespace(),
		Name:        input.GetName(),
		Key:         input.GetName(),
		FromTrigger: false,
	}

	resp := new(tester.Response)
	assert.NoError(t, EnsureAllAppsRemoved(req, resp))
	assert.Len(t, c.deleted, 2)
	assert.Equal(t, resp.Delay, 5*time.Second)
}

func TestEnsureAllParentAppsRemoved(t *testing.T) {
	input := &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-project",
		},
	}
	existing := []kclient.Object{
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-project",
			},
		},
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"acorn.io/parent-acorn-name": "my-app",
				},
				Name:      "my-other-app",
				Namespace: "my-project",
			},
		},
	}

	c := &deleteClient{
		Client: &tester.Client{
			Objects:   append(existing, input.DeepCopyObject().(kclient.Object)),
			SchemeObj: scheme.Scheme,
		},
	}
	req := router.Request{
		Client:      c,
		Object:      input,
		Ctx:         context.Background(),
		GVK:         v1.SchemeGroupVersion.WithKind("ProjectInstance"),
		Namespace:   input.GetNamespace(),
		Name:        input.GetName(),
		Key:         input.GetName(),
		FromTrigger: false,
	}

	resp := new(tester.Response)
	assert.NoError(t, EnsureAllAppsRemoved(req, resp))
	assert.Len(t, c.deleted, 1)
	assert.Equal(t, c.deleted[0].GetName(), "my-app")
	assert.Equal(t, resp.Delay, 5*time.Second)
}

func TestEnsureChildOfChildNotRemoved(t *testing.T) {
	input := &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-project",
		},
	}
	existing := []kclient.Object{
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-app",
				Namespace: "my-project",
				Labels: map[string]string{
					"acorn.io/parent-acorn-name": "already-deleted-app",
				},
			},
		},
		&v1.AppInstance{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"acorn.io/parent-acorn-name": "my-app",
				},
				Name:      "my-other-app",
				Namespace: "my-project",
			},
		},
	}

	c := &deleteClient{
		Client: &tester.Client{
			Objects:   append(existing, input.DeepCopyObject().(kclient.Object)),
			SchemeObj: scheme.Scheme,
		},
	}
	req := router.Request{
		Client:      c,
		Object:      input,
		Ctx:         context.Background(),
		GVK:         v1.SchemeGroupVersion.WithKind("ProjectInstance"),
		Namespace:   input.GetNamespace(),
		Name:        input.GetName(),
		Key:         input.GetName(),
		FromTrigger: false,
	}

	resp := new(tester.Response)
	assert.NoError(t, EnsureAllAppsRemoved(req, resp))
	assert.Len(t, c.deleted, 1)
	assert.Equal(t, "my-app", c.deleted[0].GetName())
	assert.Equal(t, 5*time.Second, resp.Delay)
}

func TestEnsureAllAppsRemoved(t *testing.T) {
	input := &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "my-project",
		},
	}
	c := &deleteClient{
		Client: &tester.Client{
			Objects:   []kclient.Object{input.DeepCopyObject().(kclient.Object)},
			SchemeObj: scheme.Scheme,
		},
	}
	req := router.Request{
		Client:      c,
		Object:      input,
		Ctx:         context.Background(),
		GVK:         v1.SchemeGroupVersion.WithKind("ProjectInstance"),
		Namespace:   input.GetNamespace(),
		Name:        input.GetName(),
		Key:         input.GetName(),
		FromTrigger: false,
	}

	resp := new(tester.Response)
	assert.NoError(t, EnsureAllAppsRemoved(req, resp))
	assert.Equal(t, resp.Delay, time.Duration(0))
}

type deleteClient struct {
	*tester.Client
	deleted []kclient.Object
}

func (c *deleteClient) Delete(_ context.Context, obj kclient.Object, _ ...kclient.DeleteOption) error {
	c.deleted = append(c.deleted, obj)
	return nil
}
