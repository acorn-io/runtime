package events

import (
	"context"
	"strconv"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apiserver/pkg/storage"
)

type eventStrategy struct {
	strategy.CompleteStrategy
}

func (s *eventStrategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	// Unmarshal custom field selectors and strip them from the list options before
	// passing to lower-level strategies (that don't support them).
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
	if q.details {
		return list, nil
	}
	for i, event := range list.Items {
		event.Details = nil
		list.Items[i] = event
	}

	return list, nil
}
