package apps

import (
	"context"
	"encoding/json"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/event"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/types"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch, clientFactory *client.Factory) rest.Storage {
	remoteResource := remote.NewRemote(&v1.AppInstance{}, c)
	strategy := translation.NewSimpleTranslationStrategy(&Translator{}, remoteResource)
	strategy = publicname.NewStrategy(strategy)
	strategy = &eventingStrategy{ // TODO(njhale): Use constructor instead
		CompleteStrategy: strategy,
		recorder:         event.NewBlockingRecorder(c), // TODO(njhale): plumb into NewStorage
	}
	validator := NewValidator(c, clientFactory)

	return stores.NewBuilder(c.Scheme(), &apiv1.App{}).
		WithCompleteCRUD(strategy).
		WithValidateUpdate(validator).
		WithValidateCreate(validator).
		WithTableConverter(tables.AppConverter).
		Build()
}

// TODO(njhale): Move the below to a separate file

type eventRecorder interface {
	Record(context.Context, *apiv1.Event) error
}

type eventingStrategy struct {
	strategy.CompleteStrategy
	recorder eventRecorder
}

const EventTypeAppSpecUpdate = "AppSpecUpdate"

func (s *eventingStrategy) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	old, err := s.Get(ctx, obj.GetNamespace(), obj.GetName())
	if err != nil {
		logrus.Warn("Failed to get old object, event recording disabled for request: %w", err)
		return s.CompleteStrategy.Update(ctx, obj)
	}

	updated, err := s.CompleteStrategy.Update(ctx, obj)
	if err != nil {
		// Return updated because CompleteStrategy.Update is a black box; i.e. we can't assume
		// updated is nil when err is non-nil.
		return updated, err
	}

	defer func() {
		eventContext, err := v1.Mapify(struct {
			Old   types.Object `json:"old"`
			Patch []byte       `json:"patch"`
		}{
			Old:   old,
			Patch: ignoreError(mergePatch(old, obj)),
		})
		if err != nil {
			logrus.Warn("Failed to generate event context, event recording disabled for request: %w", err)
			return
		}

		if err := s.recorder.Record(ctx, &apiv1.Event{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "e-", // TODO(njhale): Autogenerate names on Create.
				Namespace:    obj.GetNamespace(),
			},
			Type:     EventTypeAppSpecUpdate,
			Severity: v1.EventSeverityInfo,
			Context:  eventContext,
			Subject: v1.EventSubject{
				Kind: obj.GetObjectKind().GroupVersionKind().Kind,
				Name: obj.GetName(),
			},
			// Description: "TODO(njhale)"
			Observed: metav1.Now(),
		}); err != nil {
			logrus.Warn("Failed to record event: %w", err)
		}
	}()
	return updated, nil
}

func mergePatch(from, to any) ([]byte, error) {
	fromBytes, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	toBytes, err := json.Marshal(to)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatch.CreateMergePatch(fromBytes, toBytes)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

func ignoreError[A any](a A, err error) A {
	if err != nil {
		logrus.Warn("Ignoring error: %w", err)
	}

	return a
}
