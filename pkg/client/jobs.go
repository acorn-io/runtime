package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/fields"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) JobGet(ctx context.Context, name string) (*apiv1.Job, error) {
	job := &apiv1.Job{}
	return job, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, job)
}

func (c *DefaultClient) JobList(ctx context.Context, opts *JobListOptions) ([]apiv1.Job, error) {
	result, listOptions := &apiv1.JobList{}, &kclient.ListOptions{Namespace: c.Namespace}

	if opts != nil && opts.App != "" {
		listOptions.FieldSelector = fields.SelectorFromSet(map[string]string{"metadata.name": opts.App})
	}

	if err := c.Client.List(ctx, result, listOptions); err != nil {
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

func (c *DefaultClient) JobRestart(ctx context.Context, name string) error {
	return c.RESTClient.Post().
		Namespace(c.Namespace).
		Resource("jobs").
		Name(name).
		SubResource("restart").
		Body(&apiv1.JobRestart{}).Do(ctx).Error()
}
