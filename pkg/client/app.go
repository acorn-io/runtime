package client

import (
	"context"
	"sort"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *client) AppDelete(ctx context.Context, name string) error {
	err := c.Client.Delete(ctx, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
	})
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func (c *client) AppList(ctx context.Context) (result []App, err error) {
	var (
		apps = make(chan v1.AppInstance)
		eg   = errgroup.Group{}
	)

	c.appsForNS(ctx, &eg, c.Namespace, apps)
	waitAndClose(&eg, apps, &err)

	for app := range apps {
		result = append(result, App{
			Name:        app.Name,
			Created:     app.CreationTimestamp,
			Revision:    app.ResourceVersion,
			Labels:      app.Labels,
			Annotations: app.Annotations,
			Image:       app.Spec.Image,
			Volumes:     app.Spec.Volumes,
			Secrets:     app.Spec.Secrets,
			Status:      app.Status,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Created.After(result[j].Created.Time)
	})

	return
}
