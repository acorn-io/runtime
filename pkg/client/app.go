package client

import (
	"context"
	"sort"

	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
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

func appToApp(app v1.AppInstance) *App {
	return &App{
		Name:        app.Name,
		Created:     app.CreationTimestamp,
		Revision:    app.ResourceVersion,
		Labels:      app.Labels,
		Annotations: app.Annotations,
		Image:       app.Spec.Image,
		Volumes:     app.Spec.Volumes,
		Secrets:     app.Spec.Secrets,
		Status:      app.Status,
	}
}

func (c *client) AppGet(ctx context.Context, name string) (*App, error) {
	app := &v1.AppInstance{}
	err := c.Client.Get(ctx, client2.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if err != nil {
		return nil, err
	}

	return appToApp(*app), nil
}

func (c *client) AppList(ctx context.Context) (result []App, err error) {
	apps := &v1.AppInstanceList{}
	err = c.Client.List(ctx, apps, &client2.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	for _, app := range apps.Items {
		result = append(result, *appToApp(app))
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Created.Time == result[j].Created.Time {
			return result[i].Name < result[j].Name
		}
		return result[i].Created.After(result[j].Created.Time)
	})

	return
}

func (c *client) AppStart(ctx context.Context, name string) error {
	app := &v1.AppInstance{}
	err := c.Client.Get(ctx, client2.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if err != nil {
		return err
	}
	if app.Spec.Stop != nil && *app.Spec.Stop {
		app.Spec.Stop = new(bool)
		return c.Client.Update(ctx, app)
	}
	return nil
}

func (c *client) AppStop(ctx context.Context, name string) error {
	app := &v1.AppInstance{}
	err := c.Client.Get(ctx, client2.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if app.Spec.Stop == nil || !*app.Spec.Stop {
		app.Spec.Stop = &[]bool{true}[0]
		return c.Client.Update(ctx, app)
	}
	return nil
}
