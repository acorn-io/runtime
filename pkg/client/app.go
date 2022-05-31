package client

import (
	"context"
	"sort"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/typed"
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

func (c *client) AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error) {
	app, err := c.AppGet(ctx, name)
	if err != nil {
		return nil, err
	}

	if opts == nil {
		return app, nil
	}

	if opts.Image != "" {
		app.Spec.Image = opts.Image
	}

	app.Labels = typed.Concat(app.Labels, opts.Labels)
	app.Annotations = typed.Concat(app.Annotations, opts.Annotations)
	app.Spec.Volumes = mergeVolumes(app.Spec.Volumes, opts.Volumes)
	app.Spec.Secrets = mergeSecrets(app.Spec.Secrets, opts.Secrets)
	app.Spec.Endpoints = mergeEndpoints(app.Spec.Endpoints, opts.Endpoints)
	app.Spec.DeployParams = typed.Concat(app.Spec.DeployParams, opts.DeployParams)

	return app, c.Client.Update(ctx, app)
}

func mergeEndpoints(appEndpoints, optsEndpoints []v1.EndpointBinding) []v1.EndpointBinding {
	for _, newEndpoint := range optsEndpoints {
		found := false
		for i, existingEndpoint := range appEndpoints {
			if existingEndpoint.Target == newEndpoint.Target {
				appEndpoints[i] = newEndpoint
				found = true
				break
			}
		}
		if !found {
			appEndpoints = append(appEndpoints, newEndpoint)
		}
	}

	return appEndpoints
}

func mergeSecrets(appSecrets, optsSecrets []v1.SecretBinding) []v1.SecretBinding {
	for _, newSecret := range optsSecrets {
		found := false
		for i, existingSecret := range appSecrets {
			if existingSecret.SecretRequest == newSecret.SecretRequest {
				appSecrets[i] = newSecret
				found = true
				break
			}
		}
		if !found {
			appSecrets = append(appSecrets, newSecret)
		}
	}

	return appSecrets
}

func mergeVolumes(appVolumes, optsVolumes []v1.VolumeBinding) []v1.VolumeBinding {
	for _, newVolume := range optsVolumes {
		found := false
		for i, existingVolume := range appVolumes {
			if existingVolume.VolumeRequest == newVolume.VolumeRequest {
				appVolumes[i] = newVolume
				found = true
				break
			}
		}
		if !found {
			appVolumes = append(appVolumes, newVolume)
		}
	}

	return appVolumes
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
