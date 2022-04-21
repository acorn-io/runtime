package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/goombaio/namegenerator"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/pullsecret"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/ibuildthecloud/herd/pkg/tags"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/staging/src/k8s.io/apimachinery/pkg/api/errors"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	nameGenerator = namegenerator.NewNameGenerator(time.Now().UnixNano())
)

func (c *client) checkRemotePermissions(ctx context.Context, image string, pullSecrets []string) error {
	keyChain, err := pullsecret.Keychain(ctx, c.Client, c.Namespace, pullSecrets...)
	if err != nil {
		return err
	}

	ref, err := name.ParseReference(image)
	if err != nil {
		return err
	}

	_, err = remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(keyChain))
	if err != nil {
		return fmt.Errorf("failed to pull %s: %v", image, err)
	}
	return nil
}

func (c *client) resolveTag(ctx context.Context, image string, pullSecrets []string) (string, error) {
	localImage, err := c.ImageGet(ctx, image)
	if apierror.IsNotFound(err) {
		if tags.IsLocalReference(image) {
			return "", err
		}
		if err := c.checkRemotePermissions(ctx, image, pullSecrets); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
		return strings.TrimPrefix(localImage.Digest, "sha256:"), nil
	}
	return image, nil
}

func (c *client) AppRun(ctx context.Context, image string, opts *AppRunOptions) (*App, error) {
	if opts == nil {
		opts = &AppRunOptions{}
	}

	var (
		app     *v1.AppInstance
		lastErr error
		runOpts = run.Options{
			Name:             opts.Name,
			Namespace:        c.Namespace,
			Annotations:      opts.Annotations,
			Labels:           opts.Labels,
			Endpoints:        opts.Endpoints,
			Client:           c.Client,
			ImagePullSecrets: opts.ImagePullSecrets,
			DeployParams:     opts.DeployParams,
			Volumes:          opts.Volumes,
			Secrets:          opts.Secrets,
		}
	)

	image, err := c.resolveTag(ctx, image, opts.ImagePullSecrets)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 3; i++ {
		app, lastErr = run.Run(ctx, image, &runOpts)
		if lastErr == nil {
			return appToApp(*app), nil
		} else if apierror.IsAlreadyExists(lastErr) && opts.Name == "" {
			continue
		} else {
			return nil, lastErr
		}
	}

	return nil, fmt.Errorf("after three tried failed to create app: %w", lastErr)
}

func (c *client) AppDelete(ctx context.Context, name string) (*App, error) {
	app, err := c.AppGet(ctx, name)
	if errors.IsNotFound(err) {
		return nil, nil
	}

	return app, c.Client.Delete(ctx, &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.Namespace,
		},
	})
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
