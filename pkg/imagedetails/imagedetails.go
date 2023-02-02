package imagedetails

import (
	"context"
	"fmt"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/images"
	"github.com/acorn-io/acorn/pkg/tags"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImageDetails(ctx context.Context, c kclient.Client, namespace, imageName string, profiles []string, deployArgs map[string]any, opts ...remote.Option) (*apiv1.ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "+", "/")
	name := strings.ReplaceAll(imageName, "/", "+")

	if tagPattern, isPattern := autoupgrade.AutoUpgradePattern(imageName); isPattern {
		if latestImage, found, err := autoupgrade.FindLatestTagForImageWithPattern(ctx, c, namespace, imageName, tagPattern); err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("unable to find an image for %v matching pattern %v", imageName, tagPattern)
		} else {
			imageName = latestImage
			name = strings.ReplaceAll(imageName, "/", "+")
		}
	}

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

	appImage, err := images.PullAppImage(ctx, c, namespace, imageName, opts...)
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
