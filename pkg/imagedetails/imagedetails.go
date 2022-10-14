package imagedetails

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/pull"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImageDetails(ctx context.Context, c kclient.Client, namespace, imageName string, profiles []string, deployArgs map[string]any) (*apiv1.ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "+", "/")
	name := strings.ReplaceAll(imageName, "/", "+")

	image := &apiv1.Image{}
	err := c.Get(ctx, router.Key(namespace, name), image)
	if err != nil && !apierror.IsNotFound(err) {
		return nil, err
	} else if err != nil && apierror.IsNotFound(err) && tags.IsLocalReference(name) {
		return nil, err
	} else if err == nil {
		namespace = image.Namespace
		imageName = image.Name
	}

	appImage, err := pull.AppImage(ctx, c, namespace, imageName)
	if err != nil {
		return nil, err
	}

	details, err := ParseDetails(appImage.Acornfile, deployArgs, profiles)
	if err != nil {
		return &apiv1.ImageDetails{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imageName,
				Namespace: namespace,
			},
			ParseError: err.Error(),
		}, nil
	}

	return &apiv1.ImageDetails{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imageName,
			Namespace: namespace,
		},
		DeployArgs: details.DeployArgs,
		Profiles:   profiles,
		Params:     details.Params,
		AppSpec:    details.AppSpec,
		AppImage:   *appImage,
	}, nil
}
