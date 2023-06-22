package apps

import (
	"context"
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/jobs"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
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

	app := &apiv1.App{}
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
