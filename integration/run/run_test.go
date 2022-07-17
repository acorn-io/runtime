package run

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

func TestVolume(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	image, err := build.Build(ctx, "./testdata/volume/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/volume",
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
			obj.Labels[labels.AcornManaged] == "true"
	})

	_, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Volume:        pv.Name,
				VolumeRequest: "external",
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
			obj.Labels[labels.AcornManaged] == "true"
	})

	helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})
}

func TestImageNameAnnotation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	image, err := build.Build(helper.GetCTX(t), "./testdata/named/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})

	helper.Wait(t, client.Watch, &corev1.PodList{}, func(pod *corev1.Pod) bool {
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
		return mapping["sha256:"+digest] == "nginx"
	})
}

func TestSimple(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, ns := helper.ClientAndNamespace(t)

	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})
	assert.NotEmpty(t, app.Status.Namespace)
}

func TestDeployParam(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(ctx, "./testdata/params/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/params",
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

	appInstance, err := run.Run(helper.GetCTX(t), image.ID, &run.Options{
		Namespace: ns.Name,
		DeployArgs: map[string]interface{}{
			"someInt": 5,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})

	assert.Equal(t, "5", appInstance.Status.AppSpec.Containers["foo"].Environment[0].Value)
}

func TestPublishAcornHTTP(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(ctx, "./testdata/nested/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/nested",
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), image.ID, &run.Options{
		Namespace:        ns.Name,
		PublishProtocols: []v1.Protocol{v1.ProtocolTCP, v1.ProtocolHTTP},
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(appInstance *v1.AppInstance) bool {
		return appInstance.Status.Namespace != ""
	})

	childApp := helper.Wait(t, client.Watch, &v1.AppInstanceList{}, func(app *v1.AppInstance) bool {
		return app.Namespace == appInstance.Status.Namespace && app.Status.Namespace != ""
	})

	ingress := helper.Wait(t, client.Watch, &networkingv1.IngressList{}, func(ingress *networkingv1.Ingress) bool {
		return ingress.Namespace == childApp.Status.Namespace &&
			ingress.Name == "nginx"
	})

	assert.Equal(t, "/", ingress.Spec.Rules[0].HTTP.Paths[0].Path)
	assert.Equal(t, int32(81), ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number)
	assert.Equal(t, "nginx", ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name)
}

func TestAcornServiceExists(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(ctx, "./testdata/nested/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, ns.Name),
		Cwd:    "./testdata/nested",
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), image.ID, &run.Options{
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	_ = helper.Wait(t, client.Watch, &corev1.ServiceList{}, func(obj *corev1.Service) bool {
		if !(obj.Namespace == appInstance.Namespace &&
			obj.Name == appInstance.Name &&
			obj.Spec.Type == corev1.ServiceTypeExternalName) {
			return false
		}

		service := &corev1.Service{}
		parts := strings.Split(obj.Spec.ExternalName, ".")
		err := client.Get(ctx, router.Key(parts[1], parts[0]), service)
		if err == nil {
			assert.Len(t, service.Spec.Ports, 1)
			assert.Equal(t, int32(83), service.Spec.Ports[0].Port)
			assert.Equal(t, int32(83), service.Spec.Ports[0].TargetPort.IntVal)
			return true
		}

		return false
	})
}

func TestNested(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	image, err := build.Build(ctx, "./testdata/nested/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/nested",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, &client.AppRunOptions{
		PublishProtocols: []v1.Protocol{v1.ProtocolHTTP, v1.ProtocolTCP},
	})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Ready
	})

	helper.Wait(t, kclient.Watch, &batchv1.JobList{}, func(job *batchv1.Job) bool {
		return job.Namespace == app.Status.Namespace ||
			job.Labels[labels.AcornJobName] == "tester" &&
				job.Status.Succeeded >= 0
	})

	service := &v1.AppInstance{}
	if err := kclient.Get(ctx, router.Key(app.Status.Namespace, "service"), service); err != nil {
		t.Fatal(err)
	}

	assert.Len(t, service.Spec.Ports, 3)
	assert.False(t, service.Spec.Ports[0].Publish)
	assert.False(t, service.Spec.Ports[1].Publish)
	assert.True(t, service.Spec.Ports[2].Publish)
	assert.Equal(t, int32(83), service.Spec.Ports[2].Port)
	assert.Equal(t, int32(81), service.Spec.Ports[2].TargetPort)
}
