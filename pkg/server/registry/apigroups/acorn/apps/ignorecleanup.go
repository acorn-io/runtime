package apps

import (
	"context"
	"fmt"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/jobs"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewIgnoreCleanup(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.IgnoreCleanup{}).
		WithCreate(&ignoreCleanupStrategy{
			client: c,
		}).Build()
}

type ignoreCleanupStrategy struct {
	client client.WithWatch
}

func (s *ignoreCleanupStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	ri, _ := request.RequestInfoFrom(ctx)

	if ri.Name == "" || ri.Namespace == "" {
		return obj, nil
	}

	// Use app instance here because in Hub this request is forwarded to the workload cluster.
	// The app validation logic should not run there.
	app := &v1.AppInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, app)
	if err != nil {
		return nil, err
	}

	if app.DeletionTimestamp.IsZero() {
		return nil, fmt.Errorf("cannot force delete app %s because it is not being deleted", app.Name)
	}

	// If the app has the destroy job finalizer, remove it to force delete
	if idx := slices.Index(app.Finalizers, jobs.DestroyJobFinalizer); idx >= 0 {
		app.Finalizers = append(app.Finalizers[:idx], app.Finalizers[idx+1:]...)

		if err = s.client.Update(ctx, app); err != nil {
			return nil, err
		}
	}

	return obj, nil
}

func (s *ignoreCleanupStrategy) New() types.Object {
	return &apiv1.IgnoreCleanup{}
}
