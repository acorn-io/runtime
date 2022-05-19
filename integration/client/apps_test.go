package client

import (
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestAppStartStop(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := newImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, app.Spec.Stop)

	err = c.AppStop(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	newApp, err := c.AppGet(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, *newApp.Spec.Stop)

	err = c.AppStart(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	newApp, err = c.AppGet(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, *newApp.Spec.Stop)
}

func TestAppDelete(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := newImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	newApp, err := c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, imageID, newApp.Spec.Image)
	assert.Equal(t, app.UID, newApp.UID)

	newApp, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, newApp)
}

func TestAppGet(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := newImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	newApp, err := c.AppGet(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, imageID, newApp.Spec.Image)
	assert.Equal(t, app.UID, newApp.UID)
}

func TestAppList(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := newImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	apps, err := c.AppList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, apps, 1)
	assert.Equal(t, imageID, apps[0].Spec.Image)
	assert.Equal(t, app.UID, apps[0].UID)
}

func TestAppRun(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := newImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, &client.AppRunOptions{
		Name:        "",
		Annotations: map[string]string{"akey": "avalue"},
		Labels:      map[string]string{"lkey": "lvalue"},
		Endpoints: []v1.EndpointBinding{
			{
				Target:   "target",
				Hostname: "hostname",
			},
		},
		Volumes: []v1.VolumeBinding{
			{
				Volume:        "volume",
				VolumeRequest: "volumeRequest",
			},
		},
		Secrets: []v1.SecretBinding{
			{
				Secret:        "secret",
				SecretRequest: "secretRequest",
			},
		},
		DeployParams: map[string]interface{}{
			"key": "value",
		},
		ImagePullSecrets: []string{"pullSecret"},
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ns.Name, app.Namespace)
	assert.NotEqual(t, "", app.Name)
	assert.Equal(t, "target", app.Spec.Endpoints[0].Target)
	assert.Equal(t, "volume", app.Spec.Volumes[0].Volume)
	assert.Equal(t, "secret", app.Spec.Secrets[0].Secret)
	assert.Equal(t, "value", app.Spec.DeployParams["key"])
	assert.Equal(t, "pullSecret", app.Spec.ImagePullSecrets[0])
}
