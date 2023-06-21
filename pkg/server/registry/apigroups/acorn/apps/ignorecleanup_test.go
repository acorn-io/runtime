package apps

import (
	"context"
	"testing"
	"time"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/jobs"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	ktesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestIgnoreCleanupStrategy(t *testing.T) {
	tests := []struct {
		name                    string
		app                     *v1.AppInstance
		wantError, expectUpdate bool
	}{
		{
			name: "error if app is not deleting",
			app: &v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-app",
					Namespace: "my-project",
				},
			},
			wantError: true,
		},
		{
			name: "remove finalizer if app is deleting",
			app: &v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers:        []string{jobs.DestroyJobFinalizer},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Name:              "my-app",
					Namespace:         "my-project",
				},
			},
			expectUpdate: true,
		},
		{
			name: "remove finalizer from end if app is deleting",
			app: &v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers:        []string{"first-finalizer", "another-finalizer", jobs.DestroyJobFinalizer},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Name:              "my-app",
					Namespace:         "my-project",
				},
			},
			expectUpdate: true,
		},
		{
			name: "no update if the delete job finalizer is not present",
			app: &v1.AppInstance{
				ObjectMeta: metav1.ObjectMeta{
					Finalizers:        []string{"first-finalizer", "another-finalizer"},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
					Name:              "my-app",
					Namespace:         "my-project",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := request.WithRequestInfo(context.Background(), &request.RequestInfo{
				Name:      tt.app.Name,
				Namespace: tt.app.Namespace,
			})

			tracker := &objectTracker{
				t:             t,
				ObjectTracker: ktesting.NewObjectTracker(scheme.Scheme, kscheme.Codecs.UniversalDecoder()),
				app:           tt.app,
			}
			_, err := (&ignoreCleanupStrategy{
				client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(tt.app).WithObjectTracker(tracker).Build(),
			}).Create(ctx, &apiv1.ConfirmUpgrade{})
			if (err != nil) != tt.wantError {
				t.Errorf("ignoreCleanupStrategy.Create() error = %v, wantError %v", err, tt.wantError)
			}
			if tt.expectUpdate == (tracker.updateCalls != 1) {
				t.Errorf("ignoreCleanupStrategy.Create() updateCalls = %v, expectUpdate %v", tracker.updateCalls, tt.expectUpdate)
			}
		})
	}
}

type objectTracker struct {
	t           *testing.T
	app         *v1.AppInstance
	updateCalls int
	ktesting.ObjectTracker
}

func (o *objectTracker) Update(gvr schema.GroupVersionResource, obj runtime.Object, ns string) error {
	if app, ok := obj.(*v1.AppInstance); ok {
		o.updateCalls++
		assert.NotContains(o.t, app.Finalizers, jobs.DestroyJobFinalizer, "finalizer should be removed")
		assert.Equal(o.t, len(o.app.Finalizers)-1, len(app.Finalizers), "only job delete finalizer should be removed")
	}
	return o.ObjectTracker.Update(gvr, obj, ns)
}

func (o *objectTracker) Delete(gvr schema.GroupVersionResource, ns, name string) error {
	o.updateCalls++
	// This should only be called if the app had only the delete job finalizer.
	assert.Equal(o.t, 0, len(o.app.Finalizers)-1, "delete should only be called if the app had only the delete job finalizer")
	return o.ObjectTracker.Delete(gvr, ns, name)
}
