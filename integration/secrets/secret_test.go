package run

import (
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestText(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/generated/acorn.cue", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/generated",
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
			Labels: map[string]string{
				labels.AcornRootNamespace: ns.Name,
			},
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

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/generated-json/acorn.cue", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/generated-json",
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
			Labels: map[string]string{
				labels.AcornRootNamespace: ns.Name,
			},
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
