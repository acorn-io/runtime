package client

import (
	"context"
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/channels"
	"github.com/acorn-io/acorn/pkg/streams"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kwatch "k8s.io/apimachinery/pkg/watch"
)

func (c *DefaultClient) EventStream(ctx context.Context, opts *EventStreamOptions) (<-chan apiv1.Event, error) {
	var initial apiv1.EventList
	listOpts := opts.ListOptions()
	listOpts.Namespace = c.Namespace
	if opts.ResourceVersion == "" {
		if err := c.Client.List(ctx, &initial, listOpts); err != nil {
			return nil, err
		}

		// Set options s.t. the watch starts *after* the list
		listOpts.Raw = &metav1.ListOptions{
			ResourceVersion: initial.ResourceVersion,
		}
	}

	var (
		w   kwatch.Interface
		err error
	)
	if opts.Follow {
		w, err = c.Client.Watch(ctx, &apiv1.EventList{}, listOpts)
	}
	if err != nil {
		return nil, err
	}

	out := streams.CurrentOutput()
	result := make(chan apiv1.Event, len(initial.Items))
	go func() {
		defer close(result)

		// Send the initial set of events
		if err := channels.Send(ctx, result, initial.Items...); !channels.NilOrCanceled(err) {
			out.MustWriteErr(fmt.Errorf("failed to stream initial events for project [%s]: [%w]", c.GetProject(), err))
			return
		}

		if w == nil {
			// Following disabled
			return
		}

		defer w.Stop()

		if err := channels.ForEach(ctx, w.ResultChan(), func(e kwatch.Event) error {
			switch e.Type {
			case kwatch.Error:
				return fmt.Errorf("watch error: [%v]", e)
			case kwatch.Added:
				return channels.Send(ctx, result, *e.Object.(*apiv1.Event))
			}

			return nil
		}); !channels.NilOrCanceled(err) {
			out.MustWriteErr(fmt.Errorf("failed to stream ongoing events for project [%s]: [%w]", c.GetProject(), err))
		}
	}()

	return result, nil
}
