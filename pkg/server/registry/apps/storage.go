package apps

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory) rest.Storage {
	remoteResource := remote.NewWithTranslation(&Translator{}, &v1.AppInstance{}, &v1.AppInstanceList{}, c)
	validator := NewValidator(c, clientFactory)

	return stores.NewBuilder(c.Scheme(), &apiv1.App{}).
		WithCreate(remoteResource).
		WithGet(remoteResource).
		WithList(remoteResource).
		WithUpdate(remoteResource).
		WithDelete(remoteResource).
		WithWatch(remoteResource).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.AppConverter).
		Build()
}
