package apps

import (
	"context"
	"net/http"
	"strings"

	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/imagedetails"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewIcon(c client.WithWatch, transport http.RoundTripper) *Icon {
	return &Icon{
		client:       c,
		transportOpt: remote.WithTransport(transport),
	}
}

type Icon struct {
	*strategy.DestroyAdapter
	client       client.WithWatch
	transportOpt remote.Option
}

func (i *Icon) NamespaceScoped() bool {
	return true
}

func (i *Icon) New() runtime.Object {
	return &apiv1.IconOptions{}
}

func (i *Icon) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.IconOptions{}, false, ""
}

func (i *Icon) Connect(ctx context.Context, id string, _ runtime.Object, _ rest.Responder) (http.Handler, error) {
	ns, _ := request.NamespaceFrom(ctx)
	app := &apiv1.App{}
	err := i.client.Get(ctx, kclient.ObjectKey{Namespace: ns, Name: id}, app)
	if err != nil {
		return nil, err
	}

	pullTag, err := imagesystem.ParseAndEnsureNotInternalRepo(ctx, i.client, app.Namespace, app.Status.AppImage.ID)
	if err != nil {
		return nil, err
	}

	opts, err := images.GetAuthenticationRemoteOptionsWithLocalAuth(ctx, pullTag.Context(), nil, i.client, app.Namespace, i.transportOpt)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Downloading icon from %s (%#v)", pullTag.String(), pullTag)
	icon, suffix, err := imagedetails.GetImageIcon(ctx, i.client, app.Namespace, app.Status.AppImage.ID, app.Status.AppImage.Digest, opts...)
	if err != nil {
		return nil, err
	}

	contentType := ""
	switch strings.ToLower(suffix) {
	case ".png":
		contentType = "image/png"
	case ".svg":
		contentType = "image/svg+xml"
	case ".jpg":
		contentType = "image/jpeg"
	case ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	default:
		return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
		}), nil
	}

	if len(icon) == 0 {
		return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
			rw.WriteHeader(http.StatusNotFound)
		}), nil
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", contentType)
		_, _ = rw.Write(icon)
	}), nil
}

func (i *Icon) ConnectMethods() []string {
	return []string{"GET"}
}
