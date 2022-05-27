package client

import (
	"context"
	"sort"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *client) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error) {
	if opts == nil {
		opts = &AppRunOptions{}
	}

	var (
		app = &apiv1.App{
			ObjectMeta: metav1.ObjectMeta{
				Name:        opts.Name,
				Namespace:   c.Namespace,
				Annotations: opts.Annotations,
				Labels:      opts.Labels,
			},
			Spec: v1.AppInstanceSpec{
				Image:        image,
				Endpoints:    opts.Endpoints,
				DeployParams: opts.DeployParams,
				Volumes:      opts.Volumes,
				Secrets:      opts.Secrets,
			},
		}
	)

	return app, c.Client.Create(ctx, app)
}

func (c *client) AppDelete(ctx context.Context, name string) (*apiv1.App, error) {
	app, err := c.AppGet(ctx, name)
	if errors.IsNotFound(err) {
		return nil, nil
	}

	return app, c.Client.Delete(ctx, &apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
	})
}

func (c *client) AppGet(ctx context.Context, name string) (*apiv1.App, error) {
	app := &apiv1.App{}
	err := c.Client.Get(ctx, client2.ObjectKey{
		Name:      name,
		Namespace: c.Namespace,
	}, app)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func (c *client) AppList(ctx context.Context) ([]apiv1.App, error) {
	apps := &apiv1.AppList{}
	err := c.Client.List(ctx, apps, &client2.ListOptions{
		Namespace: c.Namespace,
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(apps.Items, func(i, j int) bool {
		if apps.Items[i].CreationTimestamp.Time == apps.Items[j].CreationTimestamp.Time {
			return apps.Items[i].Name < apps.Items[j].Name
		}
		return apps.Items[i].CreationTimestamp.After(apps.Items[j].CreationTimestamp.Time)
	})

	return apps.Items, nil
}

func (c *client) AppStart(ctx context.Context, name string) error {
	app := &apiv1.App{}
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
	app := &apiv1.App{}
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
