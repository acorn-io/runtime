package apps

import (
	"strings"
	"testing"

	client2 "github.com/acorn-io/acorn/integration/client"
	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestAppStartStop(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := client2.NewImage(t, ns.Name)

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

	imageID := client2.NewImage(t, ns.Name)

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

func TestAppUpdate(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := client2.NewImage(t, ns.Name)
	imageID2 := client2.NewImage2(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, &client.AppRunOptions{
		Annotations: map[string]string{
			"anno1": "val1",
			"anno2": "val2",
		},
		Labels: map[string]string{
			"label1": "val1",
			"label2": "val2",
		},
		Volumes: []v1.VolumeBinding{
			{
				Volume:        "vol1",
				VolumeRequest: "volreq1",
			},
			{
				Volume:        "vol2",
				VolumeRequest: "volreq2",
			},
		},
		Secrets: []v1.SecretBinding{
			{
				Secret:        "sec1",
				SecretRequest: "secreq1",
			},
			{
				Secret:        "sec2",
				SecretRequest: "secreq2",
			},
		},
		Services: []v1.ServiceBinding{
			{
				Target:  "svc-target1",
				Service: "other-service1",
			},
			{
				Target:  "svc-target2",
				Service: "other-service2",
			},
		},
		DeployArgs: map[string]interface{}{
			"param1": "val1",
			"param2": "val2",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	newApp, err := c.AppUpdate(ctx, app.Name, &client.AppUpdateOptions{
		Image: imageID2,
		Annotations: map[string]string{
			"anno2": "val3",
			"anno3": "val3",
		},
		Labels: map[string]string{
			"label2": "val3",
			"label3": "val3",
		},
		PublishMode: v1.PublishModeNone,
		Volumes: []v1.VolumeBinding{
			{
				Volume:        "vol3",
				VolumeRequest: "volreq2",
			},
			{
				Volume:        "vol3",
				VolumeRequest: "volreq3",
			},
		},
		Secrets: []v1.SecretBinding{
			{
				Secret:        "sec3",
				SecretRequest: "secreq2",
			},
			{
				Secret:        "sec3",
				SecretRequest: "secreq3",
			},
		},
		Services: []v1.ServiceBinding{
			{
				Target:  "svc-target2",
				Service: "other-service3",
			},
			{
				Target:  "svc-target3",
				Service: "other-service3",
			},
		},
		DeployArgs: map[string]interface{}{
			"param2": "val3",
			"param3": "val3",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	thirdApp, err := c.AppGet(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, newApp, thirdApp)

	assert.Equal(t, map[string]string{
		"anno1": "val1",
		"anno2": "val3",
		"anno3": "val3",
	}, thirdApp.Annotations)

	assert.Equal(t, map[string]string{
		"label1":                  "val1",
		"label2":                  "val3",
		"label3":                  "val3",
		labels.AcornRootNamespace: c.GetNamespace(),
		labels.AcornManaged:       "true",
	}, thirdApp.Labels)

	assert.Equal(t, v1.PublishModeNone, thirdApp.Spec.PublishMode)

	zero, _ := resource.ParseQuantity("0")
	assert.Equal(t, []v1.VolumeBinding{
		{
			Volume:        "vol1",
			VolumeRequest: "volreq1",
			Capacity:      zero,
		},
		{
			Volume:        "vol3",
			VolumeRequest: "volreq2",
			Capacity:      zero,
		},
		{
			Volume:        "vol3",
			VolumeRequest: "volreq3",
			Capacity:      zero,
		},
	}, thirdApp.Spec.Volumes)

	assert.Equal(t, []v1.SecretBinding{
		{
			Secret:        "sec1",
			SecretRequest: "secreq1",
		},
		{
			Secret:        "sec3",
			SecretRequest: "secreq2",
		},
		{
			Secret:        "sec3",
			SecretRequest: "secreq3",
		},
	}, thirdApp.Spec.Secrets)

	assert.Equal(t, []v1.ServiceBinding{
		{
			Target:  "svc-target1",
			Service: "other-service1",
		},
		{
			Target:  "svc-target2",
			Service: "other-service3",
		},
		{
			Target:  "svc-target3",
			Service: "other-service3",
		},
	}, thirdApp.Spec.Services)

	assert.Equal(t, v1.GenericMap{
		"param1": "val1",
		"param2": "val3",
		"param3": "val3",
	}, thirdApp.Spec.DeployArgs)

	assert.Equal(t, imageID2, thirdApp.Spec.Image)
}

func TestAppGet(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := client2.NewImage(t, ns.Name)

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

	imageID := client2.NewImage(t, ns.Name)

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

func TestAppLog(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := client2.NewImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["default"].Ready == 1
	})

	msgs, err := c.AppLog(ctx, app.Name, nil)
	if err != nil {
		t.Fatal(err)
	}

	msg1 := <-msgs
	msg2 := <-msgs

	assert.Equal(t, "", msg1.Error)
	assert.Equal(t, "", msg2.Error)
	assert.True(t, strings.HasPrefix(msg1.ContainerName, "default-"))
	assert.True(t, strings.HasPrefix(msg2.ContainerName, "default-"))
	assert.NotEqual(t, "", msg1.Line)
	assert.NotEqual(t, "", msg1.Line)

	go func() {
		for range msgs {
		}
	}()
}

func TestAppRun(t *testing.T) {
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	imageID := client2.NewImage(t, ns.Name)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, imageID, &client.AppRunOptions{
		Name:        "",
		Annotations: map[string]string{"akey": "avalue"},
		Labels:      map[string]string{"lkey": "lvalue"},
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
		DeployArgs: map[string]interface{}{
			"key": "value",
		},
		PublishMode: v1.PublishModeAll,
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ns.Name, app.Namespace)
	assert.NotEqual(t, "", app.Name)
	assert.Equal(t, v1.PublishModeAll, app.Spec.PublishMode)
	assert.Equal(t, "volume", app.Spec.Volumes[0].Volume)
	assert.Equal(t, "secret", app.Spec.Secrets[0].Secret)
	assert.Equal(t, "value", app.Spec.DeployArgs["key"])
}
