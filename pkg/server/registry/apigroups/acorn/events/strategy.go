package events

import (
	"context"
	"sort"
	"strconv"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/storage"
)

type eventStrategy struct {
	strategy.CompleteStrategy
}

func (s *eventStrategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
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
