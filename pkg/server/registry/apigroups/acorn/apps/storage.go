package apps

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/server/registry/middleware"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory, recorder event.Recorder, middlewares ...middleware.CompleteStrategy) rest.Storage {
	remoteResource := remote.NewRemote(&v1.AppInstance{}, c)
	strategy := translation.NewSimpleTranslationStrategy(&Translator{}, remoteResource)
	strategy = publicname.NewStrategy(strategy)
	strategy = newEventRecordingStrategy(strategy, recorder)
	strategy = middleware.ForCompleteStrategy(strategy, middlewares...)

	validator := NewValidator(c, clientFactory)

	return stores.NewBuilder(c.Scheme(), &apiv1.App{}).
		WithCompleteCRUD(strategy).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.AppConverter).
		Build()
}
