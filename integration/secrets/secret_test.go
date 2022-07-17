package run

import (
	"context"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
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
