package client

import (
	"context"
	"sort"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) ComputeClassGet(ctx context.Context, name string) (*apiv1.ComputeClass, error) {
	result := &apiv1.ComputeClass{}
	err := c.Client.Get(ctx, router.Key(c.Namespace, name), result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *DefaultClient) ComputeClassList(ctx context.Context) ([]apiv1.ComputeClass, error) {
	result := &apiv1.ComputeClassList{}
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
