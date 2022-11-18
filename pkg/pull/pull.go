package pull

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/pullsecret"
	"github.com/acorn-io/acorn/pkg/tags"
	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	DigestPattern = regexp.MustCompile(`^sha256:[a-f\d]{64}$`)
)

func ListTags(ctx context.Context, c client.Reader, namespace, image string) (imagename.Reference, []string, error) {
	tag, err := GetTag(ctx, c, namespace, image)
	if err != nil {
		return nil, nil, err
	}

	opts, err := GetPullOptions(ctx, c, tag, namespace)
	if err != nil {
		return nil, nil, err
	}

	tags, err := remote.List(tag.Context(), opts...)
	return tag, tags, err
}

func ImageDigest(ctx context.Context, c client.Reader, namespace, image string) (string, error) {
	tag, err := GetTag(ctx, c, namespace, image)
	if err != nil {
		return "", err
	}

	opts, err := GetPullOptions(ctx, c, tag, namespace)
	if err != nil {
		return "", err
	}

	descriptor, err := remote.Head(tag, opts...)
	if err != nil {
		return "", err
	}

	return descriptor.Digest.String(), nil
}

func AppImage(ctx context.Context, c client.Reader, namespace, image string) (*v1.AppImage, error) {
	tag, err := GetTag(ctx, c, namespace, image)
	if err != nil {
		return nil, err
	}

	opts, err := GetPullOptions(ctx, c, tag, namespace)
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

func GetTag(ctx context.Context, c client.Reader, namespace, image string) (imagename.Reference, error) {
	if tags.SHAPattern.MatchString(image) {
		port, err := buildkit.GetRegistryPort(ctx, c)
		if err != nil {
			return nil, err
		}

		image = fmt.Sprintf("127.0.0.1:%d/acorn/%s@sha256:%s", port, namespace, image)
	}
	return imagename.ParseReference(image)
}

func GetPullOptions(ctx context.Context, client client.Reader, tag imagename.Reference, namespace string) ([]remote.Option, error) {
	authn, err := pullsecret.Keychain(ctx, client, namespace)
	if err != nil {
		return nil, err
	}

	result := []remote.Option{
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(authn),
	}

	if !strings.HasPrefix(tag.Context().RegistryStr(), "127.0.0.1:") {
		return result, nil
	}

	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	port, err := buildkit.GetRegistryPort(ctx, client)
	if err != nil {
		return nil, err
	}

	if tag.Context().RegistryStr() != fmt.Sprintf("127.0.0.1:%d", port) {
		return result, nil
	}

	dialer, err := buildkit.GetRegistryDialer(ctx, c)
	if err != nil {
		return nil, err
	}

	return append(result, remote.WithTransport(&http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer(ctx, "")
		},
	})), nil
}
