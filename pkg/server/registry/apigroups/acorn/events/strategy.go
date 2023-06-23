package events

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/channels"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
)

type eventStrategy struct {
	strategy.CompleteStrategy
}

func (s *eventStrategy) Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error) {
	// Unmarshal custom field selectors and strip them from the filter options before
	// passing to lower-level strategies (that don't support them).
	q, stripped, err := stripQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to strip query from opts: [%w]", err)
	}

	events, err := s.CompleteStrategy.Watch(ctx, namespace, stripped)
	if err != nil {
		return nil, err
	}

	result := make(chan watch.Event)
	go func() {
		defer close(result)

		if err := q.filterChannel(ctx, events, result); !channels.NilOrCanceled(err) {
			logrus.Warnf("error forwarding events: [%v]", err)
		}
	}()

	return result, nil
}

func (s *eventStrategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	// Unmarshal custom field selectors and strip them from the filter options before
	// passing to lower-level strategies (that don't support them).
	q, stripped, err := stripQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to strip query from opts: [%w]", err)
	}

	unfiltered, err := s.CompleteStrategy.List(ctx, namespace, stripped)
	if err != nil {
		return nil, err
	}

	return q.filterList(unfiltered.(*apiv1.EventList)), nil
}

type query struct {
	// details determines if the details field is elided from query results.
	// If true keep details, otherwise strip them.
	details bool

	// tail when > 0, determines the number of latest events to return.
	tail int64
}

// filterChannel applies the query to every event recieved from unfiltered and forwards the result to filtered, if any.
//
// It blocks until the context is closed.
func (q query) filterChannel(ctx context.Context, unfiltered <-chan watch.Event, filtered chan<- watch.Event) error {
	return channels.ForEach(ctx, unfiltered, func(e watch.Event) error {
		fe := q.filterEvent(e)
		if fe == nil {
			// Filter out
			return nil
		}

		return channels.Send(ctx, filtered, *fe)
	})
}

// filterList applies the query to every element of list.Items and returns the result.
func (q query) filterList(list *apiv1.EventList) *apiv1.EventList {
	list.Items = q.filter(list.Items...)
	return list
}

// filterEvent applies the query to a watch.Event.
//
// It returns nil for events that don't meet the query criteria and
// a potentially modified event for those that do.
func (q query) filterEvent(e watch.Event) *watch.Event {
	switch e.Type {
	case watch.Added, watch.Modified:
	default:
		// Return unmodified
		return &e
	}

	// Attempt to filter
	obj := e.Object.(*apiv1.Event)
	filtered := q.filter(*obj)
	if len(filtered) < 1 {
		// Drop the event, it's been filtered out
		return nil
	}

	e.Object = filtered[0].DeepCopy()

	return &e
}

// filter returns the result of applying the query to a slice of events.
func (q query) filter(events ...apiv1.Event) []apiv1.Event {
	// Sort into chronological order (by observed)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Observed.Before(events[j].Observed.Time)
	})

	tail := len(events)
	if q.tail > 0 && q.tail < int64(tail) {
		tail = int(q.tail)
	}

	events = events[len(events)-tail:]

	if q.details {
		return events
	}

	for i, event := range events {
		event.Details = nil
		events[i] = event
	}

	return events
}

// stripQuery extracts the query from the given options, returning the query
// and new options sans the query.
func stripQuery(opts storage.ListOptions) (q query, stripped storage.ListOptions, err error) {
	stripped = opts

	stripped.Predicate.Field, err = stripped.Predicate.Field.Transform(func(f, v string) (string, string, error) {
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
		return
	}

	q.tail, stripped.Predicate.Limit = stripped.Predicate.Limit, 0

	return
}
