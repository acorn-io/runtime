package run

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolume(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/volume/acorn.cue", &build.Options{
		Cwd:       "./testdata/volume",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "volume-app",
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

	pv := helper.Wait(t, client.Watch, &corev1.PersistentVolumeList{}, func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == appInstance.Name &&
			obj.Labels[labels.AcornAppNamespace] == appInstance.Namespace &&
			obj.Labels[labels.AcornManaged] == "true"
	})

	err = client.Delete(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	appInstance = &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "volume-app",
			Namespace:    ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
			Volumes: []v1.VolumeBinding{
				{
					Volume:        pv.Name,
					VolumeRequest: "external",
				},
			},
		},
	}
	err = client.Create(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, client.Watch, &corev1.PersistentVolumeList{}, pv, func(obj *corev1.PersistentVolume) bool {
		return obj.Status.Phase == corev1.VolumeBound &&
			obj.Labels[labels.AcornAppName] == appInstance.Name &&
			obj.Labels[labels.AcornAppNamespace] == appInstance.Namespace &&
			obj.Labels[labels.AcornManaged] == "true"
	})

	helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})
}

func TestImageNameAnnotation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/named/acorn.cue", &build.Options{
		Cwd:       "./testdata/simple",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

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
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})

	helper.Wait(t, client.Watch, &corev1.PodList{}, func(pod *corev1.Pod) bool {
		if pod.Namespace != appInstance.Status.Namespace ||
			pod.Labels[labels.AcornAppName] != appInstance.Name ||
			pod.Annotations[labels.AcornImageMapping] == "" {
			return false
		}
		mapping := map[string]string{}
		err := json.Unmarshal([]byte(pod.Annotations[labels.AcornImageMapping]), &mapping)
		if err != nil {
			t.Fatal(err)
		}
		_, digest, _ := strings.Cut(pod.Spec.Containers[0].Image, "sha256:")
		if mapping["sha256:"+digest] == "nginx" {
			return true
		}
		return false
	})
}

func TestSimple(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/acorn.cue", &build.Options{
		Cwd:       "./testdata/simple",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

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
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})

	helper.Wait(t, client.Watch, &corev1.PodList{}, func(pod *corev1.Pod) bool {
		if pod.Namespace != appInstance.Status.Namespace ||
			pod.Labels[labels.AcornContainerName] != "tester" {
			return false
		}
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true
			}
		}
		return false
	})
}

func TestRun(t *testing.T) {
	helper.EnsureCRDs(t)

	ctx := helper.GetCTX(t)
	c := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, c)
	ns2 := ns.Name + "2"

	appInstance, err := run.Run(helper.GetCTX(t), "image", &run.Options{
		Namespace: ns2,
		Labels: map[string]string{
			"l1": "v1",
		},
		Annotations: map[string]string{
			"a1": "va1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		c.Delete(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns2,
			},
		})
	})

	assert.Equal(t, "v1", appInstance.Labels["l1"])
	assert.Equal(t, "va1", appInstance.Annotations["a1"])
	assert.Equal(t, "image", appInstance.Spec.Image)
	assert.True(t, len(appInstance.Name) > 0)
}

func TestDeployParam(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(ctx, "./testdata/params/acorn.cue", &build.Options{
		Cwd:       "./testdata/params",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	appDef, err := appdefinition.FromAppImage(image)
	if err != nil {
		t.Fatal(err)
	}

	_, err = appDef.DeployParams()
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), image.ID, &run.Options{
		Namespace: ns.Name,
		DeployParams: map[string]interface{}{
			"someInt": 5,
		},
	})

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})

	assert.Equal(t, "5", appInstance.Status.AppSpec.Containers["foo"].Environment[0].Value)
}

func TestNested(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(ctx, "./testdata/nested/acorn.cue", &build.Options{
		Cwd:       "./testdata/nested",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), image.ID, &run.Options{
		Namespace: ns.Name,
	})

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})

	helper.Wait(t, client.Watch, &batchv1.JobList{}, func(job *batchv1.Job) bool {
		if job.Namespace != appInstance.Status.Namespace ||
			job.Labels[labels.AcornJobName] != "tester" {
			return false
		}
		return job.Status.Succeeded == 1
	})
}
