package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) WorkloadClassGet(ctx context.Context, name string) (*apiv1.WorkloadClass, error) {
	result := &apiv1.WorkloadClass{}
	err := c.Client.Get(ctx, router.Key(c.Namespace, name), result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *DefaultClient) WorkloadClassList(ctx context.Context) ([]apiv1.WorkloadClass, error) {
	result := &apiv1.WorkloadClassList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{Namespace: c.Namespace})
	if err != nil {
		return nil, err
	}

	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].CreationTimestamp.Time == result.Items[j].CreationTimestamp.Time {
			return result.Items[i].Name < result.Items[j].Name
		}
		return result.Items[i].CreationTimestamp.After(result.Items[j].CreationTimestamp.Time)
	})

	return result.Items, nil
}
