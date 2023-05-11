package client

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) EventList(ctx context.Context) ([]apiv1.Event, error) {
	result := &apiv1.EventList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// func (c *DefaultClient) EventStream(ctx context.Context, opts ...ClientOption[EventStreamOptions]) (<-chan apiv1.Event, error) {
// 	// TODO(njhale): Implement me!
// 	return nil, nil
// }

func (c *DefaultClient) EventStream(ctx context.Context, opts *EventStreamOptions) (<-chan apiv1.Event, error) {
	// TODO(njhale): Implement me!
	return nil, nil
}
