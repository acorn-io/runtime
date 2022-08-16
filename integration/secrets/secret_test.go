package run

import (
	"context"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestText(t *testing.T) {
	helper.StartController(t)

	c, _ := helper.ClientAndNamespace(t)
	kclient := helper.MustReturn(k8sclient.Default)
	image, err := build.Build(helper.GetCTX(t), "./testdata/generated/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/generated",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(context.Background(), image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Namespace != ""
	})

	for _, secretName := range []string{"gen", "gen2"} {
		secret := helper.Wait(t, kclient.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
			return obj.Namespace == app.Status.Namespace &&
				obj.Name == secretName && len(obj.Data) > 0
		})
		assert.Equal(t, "static", string(secret.Data["content"]))
	}

	_, err = c.SecretGet(context.Background(), app.Name+".gen")
	if err != nil {
		t.Fatal(err)
	}

	gen2, err := c.SecretExpose(context.Background(), app.Name+".gen2")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "static", string(gen2.Data["content"]))
}

func TestJSON(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)
	client := helper.MustReturn(k8sclient.Default)

	image, err := build.Build(helper.GetCTX(t), "./testdata/generated-json/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/generated-json",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Namespace != ""
	})

	for _, secretName := range []string{"gen", "gen2"} {
		secret := helper.Wait(t, client.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
			return obj.Namespace == app.Status.Namespace &&
				obj.Name == secretName && len(obj.Data) > 0
		})
		assert.Equal(t, corev1.SecretType(v1.SecretTypePrefix+"basic"), secret.Type)
		assert.Equal(t, "value", string(secret.Data["key"]))
		assert.Equal(t, "static", string(secret.Data["pass"]))
	}
}

func TestIssue552(t *testing.T) {
	c, _ := helper.ClientAndNamespace(t)
	k8sclient := helper.MustReturn(k8sclient.Default)

	image, err := build.Build(helper.GetCTX(t), "./testdata/issue-552/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/issue-552",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(context.Background(), image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Ready &&
			app.Status.ContainerStatus["icinga2-master"].UpToDate == 1
	})

	dep := &appsv1.Deployment{}
	err = k8sclient.Get(context.Background(), router.Key(app.Status.Namespace, "icinga2-master"), dep)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(1), dep.Generation)
}
