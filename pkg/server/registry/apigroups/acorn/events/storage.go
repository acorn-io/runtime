package events

import (
	"context"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"cuelang.org/go/pkg/strconv"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/publicname"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/stores"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/sirupsen/logrus"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c kclient.WithWatch) rest.Storage {
	// TODO(njhale): Fix this up
	remoteResource := translation.NewSimpleTranslationStrategy(
		&translator{},
		remote.NewRemote(&v1.EventInstance{}, c),
	)
	strategy := &eventStrategy{
		CompleteStrategy: publicname.NewStrategy(remoteResource),
	}

	// Events are immutable, so Update is not supported.
	// Events can't be deleted directly, they are automatically GCed after
	// exceeding their TTL, so Delete is not supported.
	return stores.NewBuilder(c.Scheme(), &apiv1.Event{}).
		WithTableConverter(tables.EventConverter).
		// WithValidateCreate(&validator{}).
		WithValidateName(&validator{}).
		// TODO(njhale): Add CreateListWatch to https://github.com/acorn-io/mink/blob/9a32355ec823607b5d055aaca804d95cfcc94e95/pkg/stores/builder.go#L282
		// WithCreate(strategy).
		// WithList(strategy).
		// WithWatch(strategy).
		WithCompleteCRUD(strategy).
		Build()
}

type eventStrategy struct {
	strategy.CompleteStrategy
}

func (s *eventStrategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	logrus.Warnf("field: %s", opts.Predicate.Field.String())

	// Unmarshal custom field selectors and strip them from the list options before
	// passing to lower-level strategies (that don't support them).
	// TODO(njhale): I'm sure there's a better way to (un)marshal these
	var q query
	stripped, err := opts.Predicate.Field.Transform(func(f, v string) (string, string, error) {
		var err error
		switch f {
		case "details":
			q.details, err = strconv.ParseBool(v)
		default:
			return f, v, nil
		}

		return "", "", err
	})
	if err != nil {
		return nil, err
	}
	opts.Predicate.Field = stripped
	logrus.Warnf("query: [%v]", q)

	// TODO(njhale): Filtering like this adds an extra O(n*lgn) time and O(n) space to every List call.
	// That's not great, and might be a sign that this is the wrong level of abstraction to filter at.
	// How hard would it be to move this to the storage layer?
	full, err := s.CompleteStrategy.List(ctx, namespace, opts)
	if err != nil {
		return nil, err
	}

	return q.on(full.(*apiv1.EventList))
}

type query struct {
	// details determines if the details field is elided from query results.
	// If true keep details, otherwise strip them.
	details bool
}

func (q query) on(list *apiv1.EventList) (*apiv1.EventList, error) {
	// TODO: This can definitely be made more time-efficient and is not a "pure" function; i.e. it causes side-effects by mutating list
	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Observed.Before(&list.Items[j].Observed)
	})

	result := make([]apiv1.Event, 0, len(list.Items))
	for _, event := range list.Items {
		if len(result) == cap(result) {
			break
		}

		if !q.details {
			event.Details = nil
		}

		result = append(result, event)
	}

	list.Items = result // TODO(njhale): Will this cause an inconsistent metav1.ListMeta?

	return list, nil
}

func (s *eventStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	latest := obj.(*apiv1.Event)

	// Set the event's name to a deterministic ID generated from its content
	id, err := eventID(latest)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate event ID: [%w]", err)
	}
	latest.SetName(id)

	// TODO(njhale): Reject on max-count (either here or at a lower level; e.g. storage layer)
	created, err := s.CompleteStrategy.Create(ctx, latest)
	if err == nil {
		// Success! Bail out early
		return created, err
	}

	if !apierror.IsAlreadyExists(err) {
		return nil, err
	}

	// An event already exists with this ID, let's add an observation to it instead
	current, err := s.CompleteStrategy.Get(ctx, latest.Namespace, latest.Name)
	if err != nil {
		return nil, fmt.Errorf("Failed to get current event observation: [%w]", err)
	}

	updated, err := s.CompleteStrategy.Update(ctx, addObservation(latest, current.(*apiv1.Event)))
	if err != nil {
		return nil, fmt.Errorf("Failed to add latest event observation: [%w]", err)
	}

	return updated, nil
}

func eventID(e *apiv1.Event) (string, error) {
	// TODO: Reduce the field set used to generate when composite events are added.

	// TODO(njhale): Find a better way of selecting and encoding field sets. Maybe a multi-layered io.Writer.
	fieldSet := strings.Join([]string{
		e.Type,
		string(e.Severity),
		e.Actor,
		e.Source.String(),
		e.Description,
		e.Observed.String(),
	}, "")
	h := fnv.New128a()
	if _, err := h.Write([]byte(fieldSet)); err != nil {
		return "", err
	}

	digest := h.Sum([]byte{})

	return hex.EncodeToString(digest), nil
}

func addObservation(from, to *apiv1.Event) *apiv1.Event {
	// TODO: Implement me when composite events are added.
	return to
}
