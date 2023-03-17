package services

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(&Translator{}, &apiv1.Service{}, c)

	return stores.NewBuilder(c.Scheme(), &apiv1.Service{}).
		WithCompleteCRUD(remoteResource).
		WithTableConverter(tables.ServiceConverter).
		Build()
}
