package events

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/channels"
	"github.com/acorn-io/z"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
)

type eventStrategy struct {
	strategy.CompleteStrategy
}

func (s *eventStrategy) Create(ctx context.Context, obj types.Object) (types.Object, error) {
	return s.CompleteStrategy.Create(ctx, setDefaults(ctx, obj.(*apiv1.Event)))
}

func (s *eventStrategy) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	return s.CompleteStrategy.Update(ctx, setDefaults(ctx, obj.(*apiv1.Event)))
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

func setDefaults(ctx context.Context, e *apiv1.Event) *apiv1.Event {
	if e.Actor == "" {
		// Set actor from ctx if possible
		logrus.Debug("No Actor set, attempting to set default from request context")
		if user, ok := request.UserFrom(ctx); ok {
			e.Actor = user.GetName()
		} else {
			logrus.Debug("Request context has no user info, creating anonymous event")
		}
	}

	if e.Observed.IsZero() {
		e.Observed = internalv1.NowMicro()
	}

	return e
}

type query struct {
	// tail when > 0, determines the number of latest events to return.
	tail int64

	// prefix of an event name or source string.
	// Only events with matching names or source strings are included in query results.
	// As a special case, the empty string "" matches all events.
	prefix prefix

	// since excludes events observed before it when not nil.
	since *internalv1.MicroTime

	// until excludes events observed after it when not nil.
	until *internalv1.MicroTime
}

// filterChannel applies the query to every event received from unfiltered and forwards the result to filtered, if any.
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
// It returns nil for events that don't meet the query criteria and a potentially modified event for those that do.
func (q query) filterEvent(e watch.Event) *watch.Event {
	switch e.Type {
	case watch.Added, watch.Modified:
	default:
		// Return unmodified
		return &e
	}

	// Filter
	obj := e.Object.(*apiv1.Event)
	filtered := q.filter(*obj)
	if len(filtered) < 1 {
		// Drop the event, it's been filtered out
		return nil
	}

	e.Object = filtered[0].DeepCopy()

	return &e
}

func (q query) afterWindow(observation internalv1.MicroTime) bool {
	if q.until == nil {
		// Window includes all future events
		return false
	}

	return observation.After(q.until.Time)
}

func (q query) beforeWindow(observation internalv1.MicroTime) bool {
	if q.since == nil {
		// Window includes all existing events
		return false
	}

	return observation.Before(q.since.Time)
}

// filter returns the result of applying the query to a slice of events.
func (q query) filter(events ...apiv1.Event) []apiv1.Event {
	if len(events) < 1 {
		// Nothing to filter
		return events
	}

	// Sort into chronological order (by observed)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Observed.Before(events[j].Observed.Time)
	})

	tail := len(events)
	if q.tail > 0 && q.tail < int64(tail) {
		tail = int(q.tail)
	}

	results := make([]apiv1.Event, 0, tail)
	for _, event := range events {
		observed := event.Observed
		if q.afterWindow(observed) {
			// Exclude all events observed after the observation window ends.
			// Since the slice is sorted chronologically, we can stop filtering here.
			break
		}

		if q.beforeWindow(observed) || !q.prefix.matches(event) {
			// Exclude events:
			// - observed before the observation window starts
			// - that don't match the given prefix
			continue
		}

		results = append(results, event)
	}

	if len(results) < 1 {
		return nil
	}

	return results
}

// stripQuery extracts the query from the given options, returning the query and new options sans the query.
func stripQuery(opts storage.ListOptions) (q query, stripped storage.ListOptions, err error) {
	stripped = opts

	now := internalv1.NowMicro()
	stripped.Predicate.Field, err = stripped.Predicate.Field.Transform(func(f, v string) (string, string, error) {
		var err error
		switch f {
		case "details":
			// Detail elision is deprecated, so clients should always get details.
			// We still strip it from the selector here in order to maintain limited backwards compatibility with old
			// clients that still specify it.
		case "since":
			q.since, err = parseTimeBound(v, now, true)
		case "until":
			q.until, err = parseTimeBound(v, now, false)
		case "prefix":
			q.prefix = prefix(v)
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

// parseTimeBound parses a time bound from a string.
//
// It attempts to parse raw as one of the following formats, in order, returning the result of the first successful parse:
// 1. Go duration; e.g. "5m"
//   - time is calculated relative to now
//   - if since is true, then the duration is subtracted from now, otherwise it is added
//
// 2. RFC3339; e.g. "2006-01-02T15:04:05Z07:00"
// 3. RFC3339Micro; e.g. "2006-01-02T15:04:05.999999Z07:00"
// 4. Unix timestamp; e.g. "1136239445"
func parseTimeBound(raw string, now internalv1.MicroTime, since bool) (*internalv1.MicroTime, error) {
	// Try to parse raw as a duration string
	var errs []error
	duration, err := time.ParseDuration(raw)
	if err == nil {
		if since {
			duration *= -1
		}

		return z.P(internalv1.NewMicroTime(now.Add(duration))), nil
	}
	errs = append(errs, fmt.Errorf("%s is not a valid duration: %w", raw, err))

	// Try to parse raw as a time string
	t, err := parseTime(raw)
	if err == nil {
		return t, nil
	}
	errs = append(errs, fmt.Errorf("%s is not a valid time: %w", raw, err))

	// Try to parse raw as a unix timestamp
	unix, err := parseUnix(raw)
	if err == nil {
		return unix, nil
	}
	errs = append(errs, fmt.Errorf("%s is not a valid unix timestamp: %w", raw, err))

	return nil, errors.Join(errs...)
}

var (
	supportedLayouts = []string{
		time.RFC3339,
		metav1.RFC3339Micro,
	}
)

func parseTime(raw string) (*internalv1.MicroTime, error) {
	var errs []error
	for _, layout := range supportedLayouts {
		since, err := time.Parse(layout, raw)
		if err == nil {
			return z.P(internalv1.NewMicroTime(since)), nil
		}

		errs = append(errs, err)
	}

	return nil, errors.Join(errs...)
}

func parseUnix(raw string) (*internalv1.MicroTime, error) {
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}

	return z.P(internalv1.NewMicroTime(time.Unix(sec, 0))), nil
}

type prefix string

func (p prefix) matches(e apiv1.Event) bool {
	return p.all() ||
		strings.HasPrefix(e.Name, string(p)) ||
		strings.HasPrefix(e.Source.String(), string(p))
}

func (p prefix) all() bool {
	return p == ""
}
