package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
	result := &apiv1.ProjectList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{})
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
