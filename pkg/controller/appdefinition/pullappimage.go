package appdefinition

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	imagename "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/build/buildkit"
	"github.com/ibuildthecloud/herd/pkg/condition"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/pullsecret"
	"github.com/ibuildthecloud/herd/pkg/tags"
	"k8s.io/client-go/rest"
)

func getPullOptions(req router.Request, tag imagename.Reference, app *v1.AppInstance) ([]remote.Option, error) {
	authn, err := pullsecret.Keychain(req.Ctx, req.Client, app.Namespace, app.Spec.ImagePullSecrets...)
	if err != nil {
		return nil, err
	}

	result := []remote.Option{
		remote.WithContext(req.Ctx),
		remote.WithAuthFromKeychain(authn),
	}

	if !strings.HasPrefix(tag.Context().RegistryStr(), "127.0.0.1:") {
		return result, nil
	}

	_, err = rest.InClusterConfig()
	if err == nil {
		return result, nil
	}

	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	port, err := buildkit.GetRegistryPort(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	if tag.Context().RegistryStr() != fmt.Sprintf("127.0.0.1:%d", port) {
		return result, nil
	}

	dialer, err := buildkit.GetRegistryDialer(req.Ctx, c)
	if err != nil {
		return nil, err
	}

	return append(result, remote.WithTransport(&http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer(ctx, "")
		},
	})), nil
}

func tarToAppImage(tag imagename.Reference, reader io.Reader) (*v1.AppImage, error) {
	app, err := appdefinition.AppImageFromTar(reader)
	if err != nil {
		return nil, fmt.Errorf("invalid image %s: %v", tag, err)
	}
	return app, nil
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

	return tarToAppImage(tag, reader)
}

func getTag(req router.Request, app *v1.AppInstance) (imagename.Reference, error) {
	image := app.Spec.Image
	if tags.SHAPattern.MatchString(image) {
		port, err := buildkit.GetRegistryPort(req.Ctx, req.Client)
		if err != nil {
			return nil, err
		}

		image = fmt.Sprintf("127.0.0.1:%d/herd/%s@sha256:%s", port, app.Namespace, image)
	}
	return imagename.ParseReference(image)
}

func PullAppImage(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionPulled)

	if appInstance.Spec.Image == appInstance.Status.AppImage.ID {
		cond.Success()
		return nil
	}

	tag, err := getTag(req, appInstance)
	if err != nil {
		cond.Error(err)
		return nil
	}

	opts, err := getPullOptions(req, tag, appInstance)
	if err != nil {
		cond.Error(err)
		return nil
	}

	appImage, err := pullIndex(tag, opts)
	if err != nil {
		cond.Error(err)
		return nil
	}

	appImage.ID = appInstance.Spec.Image
	appInstance.Status.AppImage = *appImage

	cond.Success()
	return nil
}
