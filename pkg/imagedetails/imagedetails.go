package imagedetails

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
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

func GetImageIcon(ctx context.Context, c kclient.Client, namespace, imageName, digest string, opts ...remote.Option) ([]byte, string, error) {
	imageName = strings.ReplaceAll(imageName, "+", "/")
	name := strings.ReplaceAll(imageName, "/", "+")

	image := &apiv1.Image{}
	err := c.Get(ctx, router.Key(namespace, name), image)
	if kclient.IgnoreNotFound(err) != nil {
		return nil, "", err
	} else if err != nil && apierror.IsNotFound(err) && (tags.IsLocalReference(name) || tags.HasNoSpecifiedRegistry(imageName)) {
		return nil, "", err
	} else if err == nil {
		namespace = image.Namespace
		imageName = image.Name
	}

	data, err := images.PullAppImageWithDataFiles(ctx, c, namespace, imageName, digest, opts...)
	if err != nil {
		return nil, "", err
	}
	return data.Icon, data.IconSuffix, nil
}

type GetImageDetailsOptions struct {
	Profiles      []string
	DeployArgs    map[string]any
	Nested        string
	NoDefaultReg  bool
	IncludeNested bool
	RemoteOpts    []remote.Option
}

func GetImageDetails(ctx context.Context, c kclient.Client, namespace, imageName string, opts GetImageDetailsOptions) (*apiv1.ImageDetails, error) {
	remoteOpts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts.RemoteOpts...)
	if err != nil {
		return nil, err
	}

	puller, err := remote.NewPuller(remoteOpts...)
	if err != nil {
		return nil, err
	}
	opts.RemoteOpts = append(opts.RemoteOpts, remote.Reuse(puller))
	return getImageDetails(ctx, c, namespace, imageName, opts)
}

func getImageDetails(ctx context.Context, c kclient.Client, namespace, imageName string, opts GetImageDetailsOptions) (*apiv1.ImageDetails, error) {
	imageName = strings.ReplaceAll(imageName, "+", "/")
	name := strings.ReplaceAll(imageName, "/", "+")

	if tagPattern, isPattern := autoupgrade.Pattern(imageName); isPattern {
		latestImage, found, err := autoupgrade.FindLatestTagForImageWithPattern(ctx, c, "", namespace, imageName, tagPattern, opts.RemoteOpts...)
		if err != nil {
			return nil, err
		} else if !found {
			// Check and see if no registry was specified on the image.
			// If this is the case, notify the user that they need to explicitly specify docker.io if that is what they are trying to use.
			ref, err := imagename.ParseReference(strings.TrimSuffix(imageName, ":"+tagPattern), imagename.WithDefaultRegistry(images.NoDefaultRegistry))
			if err == nil && ref.Context().Registry.Name() == images.NoDefaultRegistry {
				return nil, fmt.Errorf("unable to find an image for %v matching pattern %v - if you are trying to use a remote image, specify the full registry", imageName, tagPattern)
			}

			return nil, fmt.Errorf("unable to find an image for %v matching pattern %v", imageName, tagPattern)
		}

		imageName = latestImage
		name = strings.ReplaceAll(imageName, "/", "+")
	}

	image := &apiv1.Image{}
	err := c.Get(ctx, router.Key(namespace, name), image)
	if kclient.IgnoreNotFound(err) != nil {
		return nil, err
	} else if err != nil && (tags.IsLocalReference(name) || (opts.NoDefaultReg && tags.HasNoSpecifiedRegistry(imageName))) {
		return nil, err
	} else if err == nil {
		namespace = image.Namespace
		imageName = image.Name
	}

	appImageWithData, err := images.PullAppImageWithDataFiles(ctx, c, namespace, imageName, opts.Nested, opts.RemoteOpts...)
	if err != nil {
		return nil, err
	}

	imgRef, err := images.GetImageReference(ctx, c, namespace, imageName)
	if err != nil {
		return nil, err
	}

	remoteOpts, err := images.GetAuthenticationRemoteOptions(ctx, c, namespace, opts.RemoteOpts...)
	if err != nil {
		return nil, err
	}

	_, sigHash, err := acornsign.FindSignature(imgRef.Context().Digest(appImageWithData.AppImage.Digest), remoteOpts...)
	if err != nil {
		return nil, err
	}

	details, err := ParseDetails(appImageWithData.AppImage, opts.DeployArgs, opts.Profiles)
	if err != nil {
		return &apiv1.ImageDetails{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imageName,
				Namespace: namespace,
			},
			ParseError: err.Error(),
		}, nil
	}

	permissions := getPermissions(details.AppSpec)

	var nestedImages []apiv1.NestedImage
	if opts.IncludeNested {
		nestedImages, err = getNested(ctx, c, namespace, imageName, details.AppSpec, appImageWithData.AppImage.ImageData, remoteOpts)
		if err != nil {
			return nil, err
		}
	}

	return &apiv1.ImageDetails{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appImageWithData.AppImage.Name,
			Namespace: namespace,
		},
		ImageName:       imageName,
		DeployArgs:      details.DeployArgs,
		Profiles:        opts.Profiles,
		Params:          details.Params,
		AppSpec:         details.AppSpec,
		AppImage:        *appImageWithData.AppImage,
		SignatureDigest: strings.Trim(sigHash.String(), ":"), // trim to avoid having just ':' as the digest
		Readme:          string(appImageWithData.Readme),
		Permissions:     permissions,
		NestedImages:    nestedImages,
	}, nil
}

func getNested(ctx context.Context, c kclient.Client, namespace, image string, appSpec *v1.AppSpec, imageData v1.ImagesData, remoteOpts []remote.Option) (result []apiv1.NestedImage, _ error) {
	nested, err := getNestedAcorns(ctx, c, namespace, image, appSpec, imageData, remoteOpts)
	if err != nil {
		return nil, err
	}
	result = append(result, nested...)

	nested, err = getNestedServices(ctx, c, namespace, image, appSpec, imageData, remoteOpts)
	if err != nil {
		return nil, err
	}
	result = append(result, nested...)

	return
}

// getPermissions extracts requested permissions from all containers, jobs, services and nested acorns in the app
func getPermissions(appSpec *v1.AppSpec) (result []v1.Permissions) {
	result = append(result, containerPermissions(appSpec.Containers)...)
	result = append(result, containerPermissions(appSpec.Functions)...)
	result = append(result, containerPermissions(appSpec.Jobs)...)
	result = append(result, servicePermissions(appSpec.Services)...)
	result = append(result, acornPermissions(appSpec.Acorns)...)
	return
}

func prependServiceName(serviceName string, perms []v1.Permissions) (result []v1.Permissions) {
	for _, perm := range perms {
		result = append(result, v1.Permissions{
			ServiceName: serviceName + "." + perm.ServiceName,
			Rules:       perm.GetRules(),
		})
	}
	return
}

func toNestedImage(serviceName string, details *apiv1.ImageDetails, imageName string) (result []apiv1.NestedImage) {
	result = append(result, apiv1.NestedImage{
		Name:            details.AppImage.Name,
		ImageName:       imageName,
		Digest:          details.AppImage.Digest,
		SignatureDigest: details.SignatureDigest,
		Permissions:     prependServiceName(serviceName, details.Permissions),
		ParseError:      details.ParseError,
	})

	for _, nested := range details.NestedImages {
		result = append(result, apiv1.NestedImage{
			Name:            nested.Name,
			ImageName:       nested.ImageName,
			Digest:          nested.Digest,
			SignatureDigest: nested.SignatureDigest,
			Permissions:     prependServiceName(serviceName, nested.Permissions),
			ParseError:      nested.ParseError,
		})
	}
	return
}

func getNestedAcorns(ctx context.Context, c kclient.Client, namespace, image string, app *v1.AppSpec, imageData v1.ImagesData, remoteOpts []remote.Option) (result []apiv1.NestedImage, err error) {
	for _, acornName := range typed.SortedKeys(app.Acorns) {
		acorn := app.Acorns[acornName]

		var nestedImage string
		acornImage, ok := appdefinition.GetImageReferenceForServiceName(acornName, app, imageData)
		if !ok {
			return nil, fmt.Errorf("failed to find image information for nested acorn [%s]", acornName)
		}

		if tags.IsImageDigest(acornImage) {
			nestedImage = acornImage
			acornImage = image
		}

		details, err := getImageDetails(ctx, c, namespace, acornImage, GetImageDetailsOptions{
			Profiles:      acorn.Profiles,
			DeployArgs:    acorn.DeployArgs.GetData(),
			Nested:        nestedImage,
			RemoteOpts:    remoteOpts,
			IncludeNested: true,
		})
		if err != nil {
			return nil, err
		}

		result = append(result, toNestedImage(acornName, details, acorn.Image)...)
	}

	return
}

func getNestedServices(ctx context.Context, c kclient.Client, namespace, image string, app *v1.AppSpec, imageData v1.ImagesData, remoteOpts []remote.Option) (result []apiv1.NestedImage, err error) {
	for _, serviceName := range typed.SortedKeys(app.Services) {
		service := app.Services[serviceName]

		var nestedImage string
		serviceImage, ok := appdefinition.GetImageReferenceForServiceName(serviceName, app, imageData)
		if !ok {
			// not a service acorn
			continue
		}

		if tags.IsImageDigest(serviceImage) {
			nestedImage = serviceImage
			serviceImage = image
		}

		details, err := getImageDetails(ctx, c, namespace, serviceImage, GetImageDetailsOptions{
			// Services don't have profiles
			Profiles:      nil,
			DeployArgs:    service.ServiceArgs.GetData(),
			Nested:        nestedImage,
			RemoteOpts:    remoteOpts,
			IncludeNested: true,
		})
		if err != nil {
			return nil, err
		}

		result = append(result, toNestedImage(serviceName, details, service.Image)...)
	}

	return
}

func acornPermissions(acorns map[string]v1.Acorn) (result []v1.Permissions) {
	for _, serviceName := range typed.SortedKeys(acorns) {
		acorn := acorns[serviceName]

		for _, nestedServiceName := range typed.SortedKeys(acorn.Permissions) {
			permissions := v1.Permissions{
				ServiceName: serviceName + "." + nestedServiceName,
				Rules:       acorn.Permissions[nestedServiceName].GetRules(),
			}

			if len(permissions.Rules) > 0 {
				result = append(result, permissions)
			}
		}
	}
	return
}

func servicePermissions(services map[string]v1.Service) (result []v1.Permissions) {
	for _, serviceName := range typed.SortedKeys(services) {
		service := services[serviceName]

		for _, nestedServiceName := range typed.SortedKeys(service.Permissions) {
			permissions := v1.Permissions{
				ServiceName: serviceName + "." + nestedServiceName,
				Rules:       service.Permissions[nestedServiceName].GetRules(),
			}

			if len(permissions.Rules) > 0 {
				result = append(result, permissions)
			}
		}

		if service.Consumer != nil {
			permissions := v1.Permissions{
				ServiceName: serviceName,
				Rules:       service.Consumer.Permissions.Get().GetRules(),
			}
			if len(permissions.Rules) > 0 {
				result = append(result, permissions)
			}
		}
	}
	return
}

func containerPermissions(containers map[string]v1.Container) []v1.Permissions {
	var permissions []v1.Permissions
	for _, containerName := range typed.SortedKeys(containers) {
		container := containers[containerName]
		entryPermissions := v1.Permissions{
			ServiceName: containerName,
			Rules:       container.Permissions.Get().GetRules(),
		}

		for _, sidecar := range typed.Sorted(container.Sidecars) {
			entryPermissions.Rules = append(entryPermissions.Rules, sidecar.Value.Permissions.Get().GetRules()...)
		}

		if len(entryPermissions.GetRules()) > 0 {
			permissions = append(permissions, entryPermissions)
		}
	}

	return permissions
}
