package secrets

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/tables"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewReveal(c kclient.WithWatch) rest.Storage {
	translated := translation.NewTranslationStrategy(&Translator{
		c:      c,
		reveal: true,
	}, remote.NewRemote(&corev1.Secret{}, c))
	remoteResource := publicname.NewStrategy(translated)
	return stores.NewBuilder(c.Scheme(), &apiv1.Secret{}).
		WithGet(remoteResource).
		WithTableConverter(tables.SecretConverter).
		Build()
}
