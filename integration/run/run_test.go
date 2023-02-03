package run

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolume(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/volume/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	pv := helper.Wait(t, kclient.Watch, &corev1.PersistentVolumeList{}, func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "external"
	})

	_, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Volume: pv.Name,
				Target: "external",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, kclient.Watch, &corev1.PersistentVolumeList{}, pv, func(obj *corev1.PersistentVolume) bool {
		return obj.Status.Phase == corev1.VolumeBound &&
			obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "external-bind"
	})

	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestImageNameAnnotation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	k8sclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/named/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})

	helper.Wait(t, k8sclient.Watch, &corev1.PodList{}, func(pod *corev1.Pod) bool {
		if pod.Namespace != app.Status.Namespace ||
			pod.Labels[labels.AcornAppName] != app.Name ||
			pod.Annotations[labels.AcornImageMapping] == "" {
			return false
		}
		mapping := map[string]string{}
		err := json.Unmarshal([]byte(pod.Annotations[labels.AcornImageMapping]), &mapping)
		if err != nil {
			t.Fatal(err)
		}

		_, digest, _ := strings.Cut(pod.Spec.Containers[0].Image, "sha256:")
		return mapping["sha256:"+digest] == "public.ecr.aws/docker/library/nginx:latest"
	})
}

func TestSimple(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/simple/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})
	assert.NotEmpty(t, app.Status.Namespace)
}

func TestDeployParam(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, ns := helper.ClientAndNamespace(t)
	kclient := helper.MustReturn(kclient.Default)

	image, err := c.AcornImageBuild(ctx, "./testdata/params/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/params",
	})
	if err != nil {
		t.Fatal(err)
	}

	appDef, err := appdefinition.FromAppImage(image)
	if err != nil {
		t.Fatal(err)
	}

	_, err = appDef.Args()
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), kclient, &v1.AppInstance{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
			DeployArgs: map[string]any{
				"someInt": 5,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, kclient.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})

	assert.Equal(t, "5", appInstance.Status.AppSpec.Containers["foo"].Environment[0].Value)
}
