package imagedetails

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/autoupgrade"
	acornsign "github.com/acorn-io/runtime/pkg/cosign"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/tags"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetImageDetails(ctx context.Context, c kclient.Client, namespace, imageName string, profiles []string, deployArgs map[string]any, nested string, noDefaultReg bool, opts ...remote.Option) (*apiv1.ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "+", "/")
	name := strings.ReplaceAll(imageName, "/", "+")

	if tagPattern, isPattern := autoupgrade.AutoUpgradePattern(imageName); isPattern {
		if latestImage, found, err := autoupgrade.FindLatestTagForImageWithPattern(ctx, c, "", namespace, imageName, tagPattern); err != nil {
			return nil, err
		} else if !found {
			// Check and see if no registry was specified on the image.
			// If this is the case, notify the user that they need to explicitly specify docker.io if that is what they are trying to use.
			ref, err := imagename.ParseReference(strings.TrimSuffix(imageName, ":"+tagPattern), imagename.WithDefaultRegistry(images.NoDefaultRegistry))
			if err == nil && ref.Context().Registry.Name() == images.NoDefaultRegistry {
				return nil, fmt.Errorf("unable to find an image for %v matching pattern %v - if you are trying to use a remote image, specify the full registry", imageName, tagPattern)
			}

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
	} else if err != nil && apierror.IsNotFound(err) && (tags.IsLocalReference(name) || (noDefaultReg && tags.HasNoSpecifiedRegistry(imageName))) {
		return nil, err
	} else if err == nil {
		namespace = image.Namespace
		imageName = image.Name
	}

	appImage, err := images.PullAppImage(ctx, c, namespace, imageName, nested, opts...)
	if err != nil {
		return nil, err
	}

	imgRef, err := images.GetImageReference(ctx, c, namespace, imageName)
	if err != nil {
		return nil, err
	}
	_, sigHash, err := acornsign.FindSignature(imgRef.Context().Digest(appImage.Digest), opts...)
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
		DeployArgs:      details.DeployArgs,
		Profiles:        profiles,
		Params:          details.Params,
		AppSpec:         details.AppSpec,
		AppImage:        *appImage,
		SignatureDigest: strings.Trim(sigHash.String(), ":"), // trim to avoid having just ':' as the digest
	}, nil
}
