package apps

import (
	"net/http"

	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/event"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/server/registry/middleware"
	"github.com/acorn-io/runtime/pkg/tables"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory, recorder event.Recorder, transport http.RoundTripper, middlewares ...middleware.CompleteStrategy) rest.Storage {
	remoteResource := remote.NewRemote(&v1.AppInstance{}, c)
	strategy := translation.NewSimpleTranslationStrategy(&Translator{}, remoteResource)
	strategy = publicname.NewStrategy(strategy)
	strategy = newEventRecordingStrategy(strategy, recorder)
	strategy = middleware.ForCompleteStrategy(strategy, middlewares...)

	validator := NewValidator(c, clientFactory, strategy, transport)

	return stores.NewBuilder(c.Scheme(), &apiv1.App{}).
		WithPrepareCreate(validator).
		WithCompleteCRUD(strategy).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.AppConverter).
		WithValidateName(validator).
		Build()
}
