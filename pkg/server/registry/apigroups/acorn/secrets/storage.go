package secrets

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/server/registry/middleware"
	"github.com/acorn-io/runtime/pkg/tables"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, middlewares ...middleware.CompleteStrategy) rest.Storage {
	translated := translation.NewTranslationStrategy(&Translator{
		c: c,
	}, remote.NewRemote(&corev1.Secret{}, c))
	remoteResource := publicname.NewStrategy(translated)
	remoteResource = middleware.ForCompleteStrategy(remoteResource, middlewares...)

	validator := &Validator{}

	return stores.NewBuilder(c.Scheme(), &apiv1.Secret{}).
		WithCreate(remoteResource).
		WithGet(remoteResource).
		WithList(remoteResource).
		WithUpdate(remoteResource).
		WithDelete(remoteResource).
		WithWatch(remoteResource).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.SecretConverter).
		Build()
}
