package client

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
)

func (c *DefaultClient) ImageAllowRulesGet(ctx context.Context, rulesName string) (*apiv1.ImageAllowRules, error) {
	result := &apiv1.ImageAllowRules{}
	return result, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      strings.ReplaceAll(rulesName, "/", "+"),
		Namespace: c.Namespace,
	}, result)
}

func (c *DefaultClient) ImageAllowRulesList(ctx context.Context) ([]apiv1.ImageAllowRules, error) {
	result := &apiv1.ImageAllowRulesList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}
