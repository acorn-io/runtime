package client

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
)

func (c *DefaultClient) EventStream(ctx context.Context, opts *EventStreamOptions) (<-chan apiv1.Event, error) {
	listOpts := opts.ListOptions()
	listOpts.Namespace = c.Namespace

	var current apiv1.EventList
	if err := c.Client.List(ctx, &current, listOpts); err != nil {
		return nil, err
	}

	events := make(chan apiv1.Event)
	go func() {
		defer close(events)
		// Send the current set of events
		for _, c := range current.Items {
			select {
			case <-ctx.Done():
				return
			case events <- c:
			}
		}
	}()

	return events, nil
}
