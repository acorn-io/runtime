package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) ProjectList(ctx context.Context) ([]apiv1.Project, error) {
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

func (c *DefaultClient) ProjectCreate(ctx context.Context, name string) (*apiv1.Project, error) {
	project := &apiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return project, c.Client.Create(ctx, project)
}

func (c *DefaultClient) ProjectGet(ctx context.Context, name string) (*apiv1.Project, error) {
	proj := &apiv1.Project{}
	return proj, c.Client.Get(ctx, router.Key("", name), proj)
}

func (c *DefaultClient) ProjectDelete(ctx context.Context, name string) (*apiv1.Project, error) {
	project, err := c.ProjectGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return project, c.Client.Delete(ctx, project)
}
