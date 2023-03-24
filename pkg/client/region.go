package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) RegionGet(ctx context.Context, name string) (*apiv1.Region, error) {
	region := new(apiv1.Region)
	return region, c.Client.Get(ctx, router.Key("", name), region)
}

func (c *DefaultClient) RegionList(ctx context.Context) ([]apiv1.Region, error) {
	result := new(apiv1.RegionList)
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
