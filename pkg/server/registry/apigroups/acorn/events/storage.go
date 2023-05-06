package events

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	// "github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	// TODO(njhale): Fix this up
	remoteResource := translation.NewSimpleTranslationStrategy(
		&translator{},
		remote.NewRemote(&v1.EventInstance{}, c),
	)
	// strategy := &Strategy{
	// 	delegate: publicname.NewStrategy(remoteResource),
	// }
	strategy := publicname.NewStrategy(remoteResource)

	// Events are immutable, so Update is not supported.
	// Events can't be deleted directly, they are automatically GCed after
	// exceeding their TTL, so Delete is not supported.
	return stores.NewBuilder(c.Scheme(), &apiv1.Event{}).
		WithTableConverter(tables.EventConverter).
		WithValidateCreate(&validator{}).
		// TODO(njhale): Add CreateListWatch to https://github.com/acorn-io/mink/blob/9a32355ec823607b5d055aaca804d95cfcc94e95/pkg/stores/builder.go#L282
		// WithCreate(strategy).
		// WithList(strategy).
		// WithWatch(strategy).
		WithCompleteCRUD(strategy).
		Build()
}
