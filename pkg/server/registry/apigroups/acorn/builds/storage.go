package builds

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := remote.NewWithSimpleTranslation(&Translator{}, &apiv1.AcornImageBuild{}, c)

	strategy := &Strategy{
		client:  c,
		creator: remoteResource,
	}

	return stores.NewBuilder(c.Scheme(), &apiv1.AcornImageBuild{}).
		WithCreate(strategy).
		WithGet(remoteResource).
		WithList(remoteResource).
		WithDelete(remoteResource).
		WithWatch(remoteResource).
		WithValidateCreate(strategy).
		WithTableConverter(tables.BuildConverter).
		Build()
}
