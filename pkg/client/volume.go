package client

import (
	"context"
	"sort"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *DefaultClient) VolumeList(ctx context.Context) (result []apiv1.Volume, _ error) {
	vols := &apiv1.VolumeList{}
	err := c.Client.List(ctx, vols, &kclient.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(vols.Items, func(i, j int) bool {
		if vols.Items[i].CreationTimestamp.Time == vols.Items[j].CreationTimestamp.Time {
			return vols.Items[i].Name < vols.Items[j].Name
		}
		return vols.Items[i].CreationTimestamp.After(vols.Items[j].CreationTimestamp.Time)
	})

	return vols.Items, nil
}

func (c *DefaultClient) VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error) {
	vol := &apiv1.Volume{}
	return vol, c.Client.Get(ctx, kclient.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, vol)
}

func (c *DefaultClient) VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error) {
	// get first to ensure the namespace matches
	v, err := c.VolumeGet(ctx, name)
	if apierror.IsNotFound(err) {
		return nil, nil
	}
	return v, c.Client.Delete(ctx, &apiv1.Volume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
	})
}

func (c *DefaultClient) VolumeClassList(ctx context.Context) ([]apiv1.VolumeClass, error) {
	volumeClasses := new(apiv1.VolumeClassList)
	err := c.Client.List(ctx, volumeClasses, &kclient.ListOptions{Namespace: c.Namespace})
	if err != nil {
		return nil, err
	}

	sort.Slice(volumeClasses.Items, func(i, j int) bool {
		if volumeClasses.Items[i].CreationTimestamp.Time == volumeClasses.Items[j].CreationTimestamp.Time {
			return volumeClasses.Items[i].Name < volumeClasses.Items[j].Name
		}
		return volumeClasses.Items[i].CreationTimestamp.After(volumeClasses.Items[j].CreationTimestamp.Time)
	})

	return volumeClasses.Items, nil
}

func (c *DefaultClient) VolumeClassGet(ctx context.Context, name string) (*apiv1.VolumeClass, error) {
	storage := new(apiv1.VolumeClass)
	return storage, c.Client.Get(ctx, kclient.ObjectKey{
		Namespace: c.Namespace,
		Name:      name,
	}, storage)
}
