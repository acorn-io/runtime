package images

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(
		&Translator{}, &apiv1.Image{}, c)
	strategy := NewStrategy(remoteResource, c)
	return stores.NewBuilder(c.Scheme(), &apiv1.Image{}).
		WithGet(strategy).
		WithUpdate(remoteResource).
		WithList(remoteResource).
		WithDelete(remoteResource).
		WithWatch(remoteResource).
		WithValidateUpdate(strategy).
		WithTableConverter(tables.ImageConverter).
		Build()
}
