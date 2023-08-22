package images

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/mink/pkg/validator"
	api "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagedetails"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	return imagedetails.GetImageDetails(ctx, s.client, namespace, name, imagedetails.GetImageDetailsOptions{
		RemoteOpts: []remote.Option{s.remoteOpt},
	})
}

func (s *ImageDetailStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	details := obj.(*apiv1.ImageDetails)
	if details.ImageName == "" {
		details.ImageName = details.Name
	}
	if details.ImageName == "" {
		ri, ok := request.RequestInfoFrom(ctx)
		if ok {
			details.ImageName = ri.Name
		}
	}
	ns, _ := request.NamespaceFrom(ctx)
	opts := []remote.Option{s.remoteOpt}
	imageName := strings.ReplaceAll(details.ImageName, "+", "/")
	if details.Auth != nil {
		ref, err := name.ParseReference(imageName)
		if err == nil {
			opts = append(opts, remote.WithAuthFromKeychain(images.NewSimpleKeychain(ref.Context(), *details.Auth, nil)))
		}
	}
	id, err := imagedetails.GetImageDetails(ctx, s.client, ns, details.ImageName, imagedetails.GetImageDetailsOptions{
		Profiles:      details.Profiles,
		DeployArgs:    details.DeployArgs,
		Nested:        details.NestedDigest,
		NoDefaultReg:  details.NoDefaultRegistry,
		IncludeNested: details.IncludeNested,
		RemoteOpts:    opts,
	})

	return id, translateRegistryErrors(err, imageName)
}

func (s *ImageDetailStrategy) New() types.Object {
	return &apiv1.ImageDetails{}
}

func translateRegistryErrors(in error, imageName string) error {
	if in == nil {
		return nil
	}
	if terr, ok := in.(*transport.Error); ok {
		switch terr.StatusCode {
		case http.StatusNotFound:
			return errors.NewNotFound(schema.GroupResource{Group: api.Group, Resource: "images"}, imageName)
		case http.StatusUnauthorized:
			return errors.NewUnauthorized(fmt.Sprintf("pulling image %s: %v", imageName, terr))
		case http.StatusForbidden:
			return errors.NewForbidden(schema.GroupResource{Group: api.Group, Resource: "images"}, imageName, terr)
		}
	}
	return in
}
