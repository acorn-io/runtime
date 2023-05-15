package events

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	remoteResource := translation.NewSimpleTranslationStrategy(
		&translator{},
		remote.NewRemote(&v1.EventInstance{}, c),
	)
	strategy := &eventStrategy{
		CompleteStrategy: publicname.NewStrategy(remoteResource),
	}

	return stores.NewBuilder(c.Scheme(), &apiv1.Event{}).
		WithTableConverter(tables.EventConverter).
		WithValidateName(new(validator)).
		WithCompleteCRUD(strategy).
		Build()
}
