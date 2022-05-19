package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	container := &apiv1.ContainerReplica{}
	return container, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, container)
}

func (c *client) ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error) {
	result := &apiv1.ContainerReplicaList{}
	err := c.Client.List(ctx, result, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].CreationTimestamp.Time == result.Items[j].CreationTimestamp.Time {
			return result.Items[i].Name < result.Items[j].Name
		}
		return result.Items[i].CreationTimestamp.After(result.Items[j].CreationTimestamp.Time)
	})

	if opts != nil && opts.App != "" {
		var newResult []apiv1.ContainerReplica
		for _, container := range result.Items {
			if container.Spec.AppName == opts.App {
				newResult = append(newResult, container)
			}
		}
		return newResult, nil
	}

	return result.Items, nil
}

func (c *client) ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error) {
	container, err := c.ContainerReplicaGet(ctx, name)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = c.Client.Delete(ctx, &apiv1.ContainerReplica{
		ObjectMeta: metav1.ObjectMeta{
			Name:      container.Name,
			Namespace: container.Namespace,
		},
	})
	if apierrors.IsNotFound(err) {
		return container, nil
	}
	return container, err
}
