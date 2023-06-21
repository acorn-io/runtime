package client

import (
	"context"
	"sort"

	"github.com/acorn-io/baaah/pkg/router"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
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

func (c *DefaultClient) ProjectCreate(ctx context.Context, name, defaultRegion string, supportedRegions []string) (*apiv1.Project, error) {
	project := &apiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if defaultRegion != "" {
		project.Spec.DefaultRegion = defaultRegion
		if !slices.Contains(supportedRegions, defaultRegion) {
			supportedRegions = append([]string{defaultRegion}, supportedRegions...)
		}
	}
	project.Spec.SupportedRegions = supportedRegions
	return project, c.Client.Create(ctx, project)
}

func (c *DefaultClient) ProjectUpdate(ctx context.Context, project *apiv1.Project, defaultRegion string, supportedRegions []string) (*apiv1.Project, error) {
	if defaultRegion != "" {
		project.Spec.DefaultRegion = defaultRegion
	}
	if len(supportedRegions) != 0 {
		if len(supportedRegions) == 1 && supportedRegions[0] == "" {
			// Clear supported regions
			project.Spec.SupportedRegions = nil
		} else {
			project.Spec.SupportedRegions = supportedRegions
		}
	}
	if project.Spec.DefaultRegion != "" && !slices.Contains(project.Spec.SupportedRegions, project.Spec.DefaultRegion) {
		project.Spec.SupportedRegions = append(project.Spec.SupportedRegions, project.Spec.DefaultRegion)
	}

	return project, c.Client.Update(ctx, project)
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
