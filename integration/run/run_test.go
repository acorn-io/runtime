package run

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	adminapiv1 "github.com/acorn-io/acorn/pkg/apis/admin.acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
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

	app, err = c.AppGet(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}
	assert.EqualValues(t, "1G", app.Status.AppSpec.Volumes["my-data"].Size, "volume my-data has size different than expected")

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

func TestVolumeBadClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-bad-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-bad-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, nil)
	if err == nil {
		t.Fatal("expected app with bad volume class to error on run")
	}
}

func TestVolumeBadClassInImageBoundToGoodClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping VolumeBadClassInImageBoundToGoodClass")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: storageClasses.Items[0].Name,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-bad-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-bad-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Target: "my-data",
				Class:  volumeClass.Name,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	helper.Wait(t, kclient.Watch, &corev1.PersistentVolumeClaimList{}, func(obj *corev1.PersistentVolumeClaim) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data"
	})

	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestVolumeBoundBadClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	if err := kclient.List(ctx, storageClasses); err != nil {
		t.Fatal(err)
	}
	var storageClassName string
	if len(storageClasses.Items) != 0 {
		storageClassName = storageClasses.Items[0].Name
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: storageClassName,
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-custom-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-custom-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Target: "my-data",
				Class:  "dne",
			},
		},
	})
	if err == nil {
		t.Fatal("expected app with bad volume class bound should fail to run")
	}
}

func TestVolumeClassInactive(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Inactive:   true,
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-custom-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-custom-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, nil)
	if err == nil {
		t.Fatal("expected app with inactive volume class to error on run")
	}
}

func TestVolumeClassSizeTooSmall(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Size: adminv1.VolumeClassSize{
			Min: "10Gi",
			Max: "100Gi",
		},
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-custom-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-custom-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Target: "my-data",
				Size:   "0.5Gi",
			},
		},
	})
	if err == nil {
		t.Fatal("expected app with size too small for volume class to error on run")
	}
}

func TestVolumeClassSizeTooLarge(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Size: adminv1.VolumeClassSize{
			Min: "10Gi",
			Max: "100Gi",
		},
	}
	if err := kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-custom-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-custom-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Volumes: []v1.VolumeBinding{
			{
				Target: "my-data",
				Size:   "150Gi",
			},
		},
	})
	if err == nil {
		t.Fatal("expected app with size too large for volume class to error on run")
	}
}

func TestVolumeClassRemoved(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping VolumeClassRemoved")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: storageClasses.Items[0].Name,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/volume-custom-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/volume-custom-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.Wait(t, kclient.Watch, &corev1.PersistentVolumeClaimList{}, func(obj *corev1.PersistentVolumeClaim) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data"
	})

	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		done := obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
		if done {
			if err = kclient.Delete(ctx, &volumeClass); err != nil {
				t.Fatal(err)
			}
		}

		return done
	})

	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionDefined).Error &&
			strings.Contains(obj.Status.Condition(v1.AppInstanceConditionDefined).Message, volumeClass.Name)
	})
}

func TestClusterVolumeClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ClusterVolumeClass")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: storageClasses.Items[0].Name,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/cluster-volume-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/cluster-volume-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeList), func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data"
	})

	helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestProjectVolumeClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ClusterVolumeClass")
		return
	}

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Namespace: c.GetNamespace(), Name: "acorn-test-custom"},
		StorageClassName: storageClasses.Items[0].Name,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/cluster-volume-class/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/cluster-volume-class",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeList), func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data"
	})

	helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
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

func TestDefaultClusterWorkloadClass(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	workloadClass := adminv1.ClusterWorkloadClassInstance{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Default:    true,
		CPUScaler:  0.25,
		Memory: adminv1.WorkloadClassMemory{
			Default: "512Mi",
			Max:     "1Gi",
			Min:     "512Mi",
		},
	}
	if err := kclient.Create(ctx, &workloadClass); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := kclient.Delete(context.Background(), &workloadClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

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

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
	})

	expected := map[string]v1.Scheduling{"simple": {
		Requirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi")},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("125m"),
			},
		}}}

	assert.EqualValues(t, app.Status.Scheduling, expected, "generated scheduling rules are incorrect")
}

func TestDefaultProjectWorkloadClass(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	workloadClass := adminv1.ProjectWorkloadClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-custom",
			Namespace: c.GetNamespace(),
		},
		Default:   true,
		CPUScaler: 0.25,
		Memory: adminv1.WorkloadClassMemory{
			Default: "512Mi",
			Max:     "1Gi",
			Min:     "512Mi",
		},
	}
	if err := kclient.Create(ctx, &workloadClass); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := kclient.Delete(context.Background(), &workloadClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

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

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
	})

	expected := map[string]v1.Scheduling{"simple": {
		Requirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi")},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("125m"),
			},
		}}}

	assert.EqualValues(t, app.Status.Scheduling, expected, "generated scheduling rules are incorrect")
}

func TestWorkloadClass(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	workloadClass := adminv1.ProjectWorkloadClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-custom",
			Namespace: c.GetNamespace(),
		},
		CPUScaler: 0.25,
		Memory: adminv1.WorkloadClassMemory{
			Default: "512Mi",
			Max:     "1Gi",
			Min:     "512Mi",
		},
	}
	if err := kclient.Create(ctx, &workloadClass); err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(ctx, "./testdata/workloadclass/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/workloadclass",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
	})

	expected := map[string]v1.Scheduling{"simple": {
		Requirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi")},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("125m"),
			},
		}}}

	assert.EqualValues(t, app.Status.Scheduling, expected, "generated scheduling rules are incorrect")
}

func TestUnrestrictedDefaultGetsMaximum(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	workloadClass := adminv1.ProjectWorkloadClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-custom",
			Namespace: c.GetNamespace(),
		},
		CPUScaler: 0.25,
		Memory: adminv1.WorkloadClassMemory{
			Max: "1Gi",
		},
	}
	if err := kclient.Create(ctx, &workloadClass); err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(ctx, "./testdata/workloadclass/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/workloadclass",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
	})

	expected := map[string]v1.Scheduling{"simple": {
		Requirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi")},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		}}}

	assert.EqualValues(t, app.Status.Scheduling, expected, "generated scheduling rules are incorrect")
}

func TestUnrestrictedDefaultGetsMaximumWithMinimum(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	workloadClass := adminv1.ProjectWorkloadClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "acorn-test-custom",
			Namespace: c.GetNamespace(),
		},
		CPUScaler: 0.25,
		Memory: adminv1.WorkloadClassMemory{
			Min: "512Mi",
			Max: "1Gi",
		},
	}
	if err := kclient.Create(ctx, &workloadClass); err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(ctx, "./testdata/workloadclass/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/workloadclass",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
	})

	expected := map[string]v1.Scheduling{"simple": {
		Requirements: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi")},
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("1Gi"),
				corev1.ResourceCPU:    resource.MustParse("250m"),
			},
		}}}

	assert.EqualValues(t, app.Status.Scheduling, expected, "generated scheduling rules are incorrect")
}

func TestNonExistantWorkloadClass(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	// Create acorn and intentionall do not create the WorkloadClass it references
	image, err := c.AcornImageBuild(ctx, "./testdata/workloadclass/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/workloadclass",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, nil)
	if err == nil {
		t.Fatal("expected an error to occur when creating an acorn that references a non-existant WorkloadClass")
	}
}

func TestCreateWorkloadClass(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(kclient.Default))
	kclient := helper.MustReturn(kclient.Default)
	c, err := client.New(cfg, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)

	checks := []struct {
		name      string
		memory    adminv1.WorkloadClassMemory
		cpuScaler float64
		fail      bool
	}{
		{
			name: "invalid-memory-default",
			memory: adminv1.WorkloadClassMemory{
				Default: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-min",
			memory: adminv1.WorkloadClassMemory{
				Min: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-max",
			memory: adminv1.WorkloadClassMemory{
				Max: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-values",
			memory: adminv1.WorkloadClassMemory{
				Values: []string{"invalid"},
			},
			fail: true,
		},
		{
			name: "valid-only-max",
			memory: adminv1.WorkloadClassMemory{
				Max: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-only-min",
			memory: adminv1.WorkloadClassMemory{
				Min: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-only-default",
			memory: adminv1.WorkloadClassMemory{
				Default: "512Mi",
			},
			fail: false,
		},
		{
			name: "invalid-default-less-than-min",
			memory: adminv1.WorkloadClassMemory{
				Default: "128Mi",
				Min:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-default-greater-than-max",
			memory: adminv1.WorkloadClassMemory{
				Default: "1Gi",
				Max:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-min-max-swapped",
			memory: adminv1.WorkloadClassMemory{
				Min: "1Gi",
				Max: "512Mi",
			},
			fail: true,
		},
		{
			name:      "valid-full",
			cpuScaler: 0.25,
			memory: adminv1.WorkloadClassMemory{
				Default: "512Mi",
				Max:     "2Gi",
				Min:     "128Mi",
				Values:  []string{"1Gi", "2Gi"},
			},
		},
		{
			name: "valid-empty",
		},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			// Create a non-instanced WorkloadClass to trigger Mink valdiation
			workloadClass := adminapiv1.ProjectWorkloadClass{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "acorn-test-custom",
					Namespace:    c.GetNamespace(),
				},
				CPUScaler: tt.cpuScaler,
				Memory:    tt.memory,
			}

			// TODO - dry run
			err = kclient.Create(ctx, &workloadClass)
			if err != nil && !tt.fail {
				t.Fatal("did not expect creation to fail:", err)
			} else if err == nil && tt.fail {
				if err := kclient.Delete(context.Background(), &workloadClass); err != nil && !apierrors.IsNotFound(err) {
					t.Fatal("failed to cleanup test:", err)
				}
				if tt.fail {
					t.Fatal("expected an error to occur when creating an invalid WorkloadClass but did not receive one")
				}
			}

		})
	}

}
