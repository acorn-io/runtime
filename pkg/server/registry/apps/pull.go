package apps

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewPullAppImage(c client.WithWatch) rest.Storage {
	return stores.NewBuilder(c.Scheme(), &apiv1.AppPullImage{}).
		WithCreate(&PullAppImageStrategy{
			client: c,
		}).Build()
}

type PullAppImageStrategy struct {
	client client.WithWatch
}

func (s *PullAppImageStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	p := obj.(*apiv1.AppPullImage)
	ri, _ := request.RequestInfoFrom(ctx)

	app := &v1.AppInstance{}
	err := s.client.Get(ctx, kclient.ObjectKey{Namespace: ri.Namespace, Name: ri.Name}, app)
	if err != nil {
		return nil, err
	}
	if _, pattern := autoupgrade.AutoUpgradePattern(app.Spec.Image); pattern {
		if app.Status.AppImage.Name != "" {
			app.Status.AvailableAppImage = app.Status.AppImage.Name
		}
	} else {
		app.Status.AvailableAppImage = app.Spec.Image
	}

	err = s.client.Status().Update(ctx, app)
	return p, err
}

func (s *PullAppImageStrategy) New() types.Object {
	return &apiv1.AppPullImage{}
}
