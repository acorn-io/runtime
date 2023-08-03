package images

import (
	"net/http"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, transport http.RoundTripper) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{},
		remote.NewRemote(&v1.ImageInstance{}, c))

	strategy := NewStrategy(remoteResource, c, transport)
	return stores.NewBuilder(c.Scheme(), &apiv1.Image{}).
		WithGet(strategy).
		WithUpdate(remoteResource).
		WithList(remoteResource).
		WithDelete(strategy).
		WithWatch(remoteResource).
		WithValidateUpdate(strategy).
		WithTableConverter(tables.ImageConverter).
		Build()
}
