package images

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/pull"
	"github.com/acorn-io/acorn/pkg/tags"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewImageDetails(c client.WithWatch, images *Storage) *ImageDetails {
	return &ImageDetails{
		client: c,
		images: images,
	}
}

type ImageDetails struct {
	images *Storage
	client client.WithWatch
}

func (s *ImageDetails) NamespaceScoped() bool {
	return true
}

func (s *ImageDetails) New() runtime.Object {
	return &apiv1.ImageDetails{}
}

func (s *ImageDetails) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			return nil, err
		}
	}

	details := obj.(*apiv1.ImageDetails)
	if details.Name == "" {
		ri, ok := request.RequestInfoFrom(ctx)
		if ok {
			details.Name = ri.Name
		}
	}
	return s.GetDetails(ctx, details.Name, details.Profiles, details.DeployArgs)
}

func (s *ImageDetails) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	name = strings.ReplaceAll(name, "+", "/")
	return s.GetDetails(ctx, name, nil, nil)
}

func (s *ImageDetails) GetDetails(ctx context.Context, name string, profiles []string, deployArgs map[string]interface{}) (*apiv1.ImageDetails, error) {
	ns, _ := request.NamespaceFrom(ctx)
	imageName := name

	image, err := s.images.ImageGet(ctx, name)
	if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	} else if err != nil && apierror.IsNotFound(err) && tags.IsLocalReference(name) {
		return nil, err
	} else if err == nil {
		ns = image.Namespace
		imageName = image.Name
	}

	appImage, err := pull.AppImage(ctx, s.client, ns, imageName)
	if err != nil {
		return nil, err
	}

	result := &apiv1.ImageDetails{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageName,
			Namespace: ns,
		},
		AppImage: *appImage,
	}

	appDef, err := appdefinition.NewAppDefinition([]byte(appImage.Acornfile))
	if err != nil {
		result.ParseError = err.Error()
		return result, nil
	}

	if len(deployArgs) > 0 || len(profiles) > 0 {
		appDef, deployArgs, err = appDef.WithDeployArgs(deployArgs, profiles)
		if err != nil {
			result.ParseError = err.Error()
			return result, nil
		}
		result.DeployArgs = deployArgs
	}

	appSpec, err := appDef.AppSpec()
	if err != nil {
		result.ParseError = err.Error()
		return result, nil
	}

	result.AppSpec = appSpec

	paramSpec, err := appDef.DeployParams()
	if err != nil {
		result.ParseError = err.Error()
		return result, nil
	}

	result.Params = paramSpec
	return result, nil
}
