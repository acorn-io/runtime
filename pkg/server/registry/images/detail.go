package images

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/imagedetails"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageDetails(c client.WithWatch) rest.Storage {
	strategy := &ImageDetailStrategy{client: c}
	return stores.NewBuilder(c.Scheme(), &apiv1.ImageDetails{}).
		WithGet(strategy).
		WithCreate(strategy).
		Build()
}

type ImageDetailStrategy struct {
	client client.WithWatch
}

func (s *ImageDetailStrategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return s.GetDetails(ctx, namespace, name, nil, nil)
}

func (s *ImageDetailStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	details := obj.(*apiv1.ImageDetails)
	if details.Name == "" {
		ri, ok := request.RequestInfoFrom(ctx)
		if ok {
			details.Name = ri.Name
		}
	}
	ns, _ := request.NamespaceFrom(ctx)
	return s.GetDetails(ctx, ns, details.Name, details.Profiles, details.DeployArgs)
}

func (s *ImageDetailStrategy) New() types.Object {
	return &apiv1.ImageDetails{}
}

func (s *ImageDetailStrategy) GetDetails(ctx context.Context, namespace, name string, profiles []string, deployArgs map[string]any) (*apiv1.ImageDetails, error) {
	return imagedetails.GetImageDetails(ctx, s.client, namespace, name, profiles, deployArgs)
}
