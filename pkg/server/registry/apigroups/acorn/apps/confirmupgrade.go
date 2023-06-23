package apps

import (
	"context"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewConfirmUpgrade(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.ConfirmUpgrade{}).
		WithCreate(&ConfirmUpgradeStrategy{
			client: c,
		}).Build()
}

type ConfirmUpgradeStrategy struct {
	client client.WithWatch
}

func (s *ConfirmUpgradeStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
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
	app.Status.AvailableAppImage = app.Status.ConfirmUpgradeAppImage
	app.Status.AvailableAppImageRemote = app.Status.ConfirmUpgradeAppImageRemote

	err = s.client.Status().Update(ctx, app)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (s *ConfirmUpgradeStrategy) New() types.Object {
	return &apiv1.ConfirmUpgrade{}
}
