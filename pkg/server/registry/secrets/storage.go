package secrets

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithTranslation(&Translator{
		c: c,
	}, &corev1.Secret{}, &corev1.SecretList{}, c)
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
