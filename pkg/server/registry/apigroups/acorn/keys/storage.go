package keys

import (
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(&Translator{}, remote.NewRemote(&internalv1.PublicKeyInstance{}, c))

	validator := &Validator{}
	return stores.NewBuilder(c.Scheme(), &apiv1.PublicKey{}).
		WithCompleteCRUD(remoteResource).
		WithValidateCreate(validator).
		WithValidateUpdate(validator).
		WithTableConverter(tables.PublicKeyConverter).
		Build()
}
