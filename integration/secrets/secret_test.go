package run

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	plainTextData = "plain text"
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

	gen2, err := c.SecretReveal(context.Background(), app.Name+".gen2")
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

func TestEncryptionEndToEnd(t *testing.T) {
	c1, _ := helper.ClientAndNamespace(t)
	k8sclient := helper.MustReturn(k8sclient.Default)

	info, err := c1.Info(helper.GetCTX(t))
	if err != nil {
		t.Fatal(err)
	}

	keyBytes, err := base64.RawURLEncoding.DecodeString(info.Spec.PublicKeys[0].KeyID)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, keyBytes, 32)

	encData, err := nacl.MultipleKeyEncrypt(plainTextData, []string{info.Spec.PublicKeys[0].KeyID})
	if err != nil {
		t.Fatal(err)
	}
	output, err := encData.Marshal()
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.HasPrefix(output, "ACORNENC:"))

	image, err := build.Build(helper.GetCTX(t), "./testdata/encryption/Acornfile", &build.Options{
		Client: c1,
		Cwd:    "./testdata/encryption",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c1.AppRun(context.Background(), image.ID, &client.AppRunOptions{
		DeployArgs: map[string]any{
			"encdata": output,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c1.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Namespace != ""
	})

	secret := helper.Wait(t, k8sclient.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
		return obj.Namespace == app.Status.Namespace &&
			obj.Name == "test" && len(obj.Data) > 0
	})

	assert.Equal(t, plainTextData, string(secret.Data["key"]))

}

func TestNamespacedDecryption(t *testing.T) {
	c1, _ := helper.ClientAndNamespace(t)
	c2, _ := helper.ClientAndNamespace(t)

	encdata := helper.EncryptData(t, c1, nil, plainTextData)

	image, err := build.Build(helper.GetCTX(t), "./testdata/encryption/Acornfile", &build.Options{
		Client: c2,
		Cwd:    "./testdata/encryption",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c2.AppRun(context.Background(), image.ID, &client.AppRunOptions{
		DeployArgs: map[string]any{
			"encdata": encdata,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c2.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Namespace != ""
	})

	assert.True(t, app.Status.Condition(v1.AppInstanceConditionSecrets).Error)
	assert.Equal(t, "errored: [test: No encryption keys were found]", app.Status.Condition(v1.AppInstanceConditionSecrets).Message)

}

func TestMultiKeyDecryptionEndToEnd(t *testing.T) {
	c1, _ := helper.ClientAndNamespace(t)
	c2, _ := helper.ClientAndNamespace(t)
	k8sclient := helper.MustReturn(k8sclient.Default)

	keys := helper.GetEncryptionKeys(t, []client.Client{c1, c2})
	assert.Len(t, keys, 2)

	encdata := helper.EncryptData(t, nil, keys, plainTextData)
	assert.True(t, strings.HasPrefix(encdata, "ACORNENC:"))

	image, err := build.Build(helper.GetCTX(t), "./testdata/encryption/Acornfile", &build.Options{
		Client: c1,
		Cwd:    "./testdata/encryption",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c1.AppRun(context.Background(), image.ID, &client.AppRunOptions{
		DeployArgs: map[string]any{
			"encdata": encdata,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c1.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Namespace != ""
	})

	secret := helper.Wait(t, k8sclient.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
		return obj.Namespace == app.Status.Namespace &&
			obj.Name == "test" && len(obj.Data) > 0
	})

	assert.Equal(t, plainTextData, string(secret.Data["key"]))
}
