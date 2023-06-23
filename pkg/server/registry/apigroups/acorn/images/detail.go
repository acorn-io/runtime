package images

import (
	"context"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagedetails"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageDetails(c client.WithWatch, transport http.RoundTripper) rest.Storage {
	strategy := &ImageDetailStrategy{
		client:    c,
		remoteOpt: remote.WithTransport(transport),
	}
	return stores.NewBuilder(c.Scheme(), &apiv1.ImageDetails{}).
		WithValidateName(validator.NoValidation).
		WithGet(strategy).
		WithCreate(strategy).
		Build()
}

type ImageDetailStrategy struct {
	client    client.WithWatch
	remoteOpt remote.Option
}

func (s *ImageDetailStrategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	return imagedetails.GetImageDetails(ctx, s.client, namespace, name, nil, nil, "", s.remoteOpt)
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
	opts := []remote.Option{s.remoteOpt}
	if details.Auth != nil {
		imageName := strings.ReplaceAll(details.Name, "+", "/")
		ref, err := name.ParseReference(imageName)
		if err == nil {
			opts = append(opts, remote.WithAuthFromKeychain(images.NewSimpleKeychain(ref.Context(), *details.Auth, nil)))
		}
	}
	return imagedetails.GetImageDetails(ctx, s.client, ns, details.Name, details.Profiles, details.DeployArgs, details.NestedDigest, opts...)
}

func (s *ImageDetailStrategy) New() types.Object {
	return &apiv1.ImageDetails{}
}
