package run

import (
	"testing"

	"github.com/ibuildthecloud/herd/integration/helper"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/build"
	hclient "github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestText(t *testing.T) {
	helper.StartController(t)

	image, err := build.Build(helper.GetCTX(t), "./testdata/generated/herd.cue", &build.Options{
		Cwd: "./testdata/generated",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)
	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
		},
	}

	err = client.Create(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Namespace != ""
	})

	secret := helper.Wait(t, client.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
		return obj.Namespace == appInstance.Status.Namespace &&
			obj.Name == "gen" && len(obj.Data) > 0
	})
	assert.Equal(t, "static", string(secret.Data["content"]))
}

func TestJSON(t *testing.T) {
	helper.StartController(t)

	image, err := build.Build(helper.GetCTX(t), "./testdata/generated-json/herd.cue", &build.Options{
		Cwd: "./testdata/generated-json",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)
	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
		},
	}

	err = client.Create(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Namespace != ""
	})

	secret := helper.Wait(t, client.Watch, &corev1.SecretList{}, func(obj *corev1.Secret) bool {
		return obj.Namespace == appInstance.Status.Namespace &&
			obj.Name == "gen" && len(obj.Data) > 0
	})
	assert.Equal(t, corev1.SecretType("other"), secret.Type)
	assert.Equal(t, "value", string(secret.Data["key"]))
	assert.Equal(t, "static", string(secret.Data["pass"]))
}
