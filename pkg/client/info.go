package client

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) Info(ctx context.Context) ([]apiv1.Info, error) {
	result := &apiv1.InfoList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}
	var infoList []apiv1.Info
	for _, subInfo := range result.Items {
		infoList = append(infoList, subInfo)
	}
	return infoList, nil
}
