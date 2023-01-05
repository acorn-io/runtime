package images

import (
	"context"
	"fmt"
	"regexp"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/imagesystem"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DigestPattern = regexp.MustCompile(`^sha256:[a-f\d]{64}$`)
)

func ListTags(ctx context.Context, c client.Reader, namespace, image string, opts ...remote.Option) (imagename.Reference, []string, error) {
	tag, err := GetImageReference(ctx, c, namespace, image)
	if err != nil {
		return nil, nil, err
	}

	opts, err = GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return nil, nil, err
	}

	tags, err := remote.List(tag.Context(), opts...)
	return tag, tags, err
}

func ImageDigest(ctx context.Context, c client.Reader, namespace, image string, opts ...remote.Option) (string, error) {
	tag, err := GetImageReference(ctx, c, namespace, image)
	if err != nil {
		return "", err
	}

	opts, err = GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return "", err
	}

	descriptor, err := remote.Head(tag, opts...)
	if err != nil {
		return "", err
	}

	return descriptor.Digest.String(), nil
}

func PullAppImage(ctx context.Context, c client.Reader, namespace, image string, opts ...remote.Option) (*v1.AppImage, error) {
	tag, err := GetImageReference(ctx, c, namespace, image)
	if err != nil {
		return nil, err
	}

	opts, err = GetAuthenticationRemoteOptions(ctx, c, namespace, opts...)
	if err != nil {
		return nil, err
	}

	appImage, err := pullIndex(tag, opts)
	if err != nil {
		return nil, err
	}

	appImage.ID = image
	return appImage, nil
}

func ResolveTag(tag imagename.Reference, image string) string {
	if DigestPattern.MatchString(image) {
		return tag.Context().Digest(image).String()
	}
	return image
}

func pullIndex(tag imagename.Reference, opts []remote.Option) (*v1.AppImage, error) {
	img, err := remote.Index(tag, opts...)
	if err != nil {
		return nil, err
	}

	manifest, err := img.IndexManifest()
	if err != nil {
		return nil, err
	}

	if len(manifest.Manifests) == 0 {
		return nil, fmt.Errorf("invalid manifest for %s, no manifest descriptors", tag)
	}

	image, err := img.Image(manifest.Manifests[0].Digest)
	if err != nil {
		return nil, err
	}

	layers, err := image.Layers()
	if err != nil {
		return nil, err
	}

	if len(layers) == 0 {
		return nil, fmt.Errorf("invalid image for %s, no layers", tag)
	}

	reader, err := layers[0].Uncompressed()
	if err != nil {
		return nil, err
	}

	app, err := appdefinition.AppImageFromTar(reader)
	if err != nil {
		return nil, fmt.Errorf("invalid image %s: %v", tag, err)
	}

	digest, err := img.Digest()
	if err != nil {
		return nil, err
	}
	app.Digest = digest.String()
	return app, nil
}

// GetRuntimePullableImageReference is similar to GetImageReference but will return 127.0.0.1:NODEPORT instead of
// registry.acorn-image-system.svc.cluster.local:5000, only use this method if you are passing the
// image string to a PodSpec that will be pulled by the container runtime, otherwise use GetImageReference if you will
// be pulling the image from the apiserver/controller
func GetRuntimePullableImageReference(ctx context.Context, c client.Reader, namespace, image string) (imagename.Reference, error) {
	if tags.SHAPattern.MatchString(image) {
		return imagesystem.GetRuntimePullableInternalRepoForNamespaceAndID(ctx, c, namespace, image)
	}

	return imagesystem.ParseAndEnsureNotInternalRepo(ctx, c, image)
}

func GetImageReference(ctx context.Context, c client.Reader, namespace, image string) (imagename.Reference, error) {
	if tags.SHAPattern.MatchString(image) {
		return imagesystem.GetInternalRepoForNamespaceAndID(ctx, c, namespace, image)
	}
	return imagename.ParseReference(image)
}

func GetAuthenticationRemoteOptions(ctx context.Context, client client.Reader, namespace string, additionalOpts ...remote.Option) ([]remote.Option, error) {
	authn, err := pullsecret.Keychain(ctx, client, namespace)
	if err != nil {
		return nil, err
	}

	result := []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn),
	}

	return append(result, additionalOpts...), nil
}
