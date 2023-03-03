package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/restconfig"
	k8slabels "k8s.io/apimachinery/pkg/labels"
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
	"github.com/acorn-io/acorn/pkg/tolerations"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
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
		StorageClassName: getStorageClassName(t, storageClasses),
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
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
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

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: getStorageClassName(t, storageClasses),
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
		StorageClassName: getStorageClassName(t, storageClasses),
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
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
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
		ObjectMeta:         metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName:   getStorageClassName(t, storageClasses),
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("5G"),
			Min:     v1.Quantity("1G"),
			Max:     v1.Quantity("9G"),
		},
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

	pv := helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeList), func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
	})

	assert.Equal(t, []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"}, pv.Spec.AccessModes)
	assert.Equal(t, "5G", pv.Spec.Capacity.Storage().String())

	helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestClusterVolumeClassValuesInAcornfile(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ClusterVolumeClassValuesInAcornfile")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:         metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName:   getStorageClassName(t, storageClasses),
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce, v1.AccessModeReadWriteMany},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("5G"),
			Min:     v1.Quantity("1G"),
			Max:     v1.Quantity("9G"),
		},
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/cluster-volume-class-with-values/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/cluster-volume-class-with-values",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	pvc := helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeClaimList), func(obj *corev1.PersistentVolumeClaim) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
	})

	assert.Equal(t, []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}, pvc.Spec.AccessModes)
	assert.Equal(t, "3G", pvc.Spec.Resources.Requests.Storage().String())
	// Depending on the storage class available, readWriteMany may not be supported. Don't wait for the app to deploy successfully because it may not.
}

func TestProjectVolumeClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ProjectVolumeClass")
		return
	}

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta:         metav1.ObjectMeta{Namespace: c.GetNamespace(), Name: "acorn-test-custom"},
		StorageClassName:   getStorageClassName(t, storageClasses),
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("2G"),
			Min:     v1.Quantity("1G"),
			Max:     v1.Quantity("3G"),
		},
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

	pv := helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeList), func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
	})

	assert.Equal(t, []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"}, pv.Spec.AccessModes)
	assert.Equal(t, "2G", pv.Spec.Capacity.Storage().String())

	helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestProjectVolumeClassDefaultSizeValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ProjectVolumeClassDefaultSizeValidation")
		return
	}

	storageClassName := getStorageClassName(t, storageClasses)

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta:         metav1.ObjectMeta{Namespace: c.GetNamespace(), Name: "acorn-test-custom"},
		StorageClassName:   storageClassName,
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("5G"),
			Min:     v1.Quantity("1G"),
			Max:     v1.Quantity("9G"),
		},
		Default: true,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/no-class-with-values/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/no-class-with-values",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	pv := helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeList), func(obj *corev1.PersistentVolume) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
	})

	assert.Equal(t, storageClassName, pv.Spec.StorageClassName)
	assert.Equal(t, []corev1.PersistentVolumeAccessMode{"ReadWriteOnce"}, pv.Spec.AccessModes)
	assert.Equal(t, "6G", pv.Spec.Capacity.Storage().String())

	helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
			obj.Status.Condition(v1.AppInstanceConditionVolumes).Success
	})
}

func TestProjectVolumeClassDefaultSizeBadValidation(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ProjectVolumeClassDefaultSizeBadValidation")
		return
	}

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta:         metav1.ObjectMeta{Namespace: c.GetNamespace(), Name: "acorn-test-custom"},
		StorageClassName:   getStorageClassName(t, storageClasses),
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("4G"),
			Min:     v1.Quantity("1G"),
			Max:     v1.Quantity("5G"),
		},
		Default: true,
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/no-class-with-values/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/no-class-with-values",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, nil)
	if err == nil {
		t.Fatal("expected app with size too large for volume class to error on run")
	}
}

func TestProjectVolumeClassValuesInAcornfile(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndNamespace(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kclient.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping ProjectVolumeClassValuesInAcornfile")
		return
	}

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta:         metav1.ObjectMeta{Namespace: c.GetNamespace(), Name: "acorn-test-custom"},
		StorageClassName:   getStorageClassName(t, storageClasses),
		AllowedAccessModes: []v1.AccessMode{v1.AccessModeReadWriteOnce, v1.AccessModeReadWriteMany},
		Size: adminv1.VolumeClassSize{
			Default: v1.Quantity("5G"),
			Min:     v1.Quantity("2G"),
			Max:     v1.Quantity("6G"),
		},
	}
	if err = kclient.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kclient.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	image, err := c.AcornImageBuild(ctx, "./testdata/cluster-volume-class-with-values/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/cluster-volume-class-with-values",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	pvc := helper.Wait(t, kclient.Watch, new(corev1.PersistentVolumeClaimList), func(obj *corev1.PersistentVolumeClaim) bool {
		return obj.Labels[labels.AcornAppName] == app.Name &&
			obj.Labels[labels.AcornAppNamespace] == app.Namespace &&
			obj.Labels[labels.AcornManaged] == "true" &&
			obj.Labels[labels.AcornVolumeName] == "my-data" &&
			obj.Labels[labels.AcornVolumeClass] == volumeClass.Name
	})

	assert.Equal(t, []corev1.PersistentVolumeAccessMode{"ReadWriteMany"}, pvc.Spec.AccessModes)
	assert.Equal(t, "3G", pvc.Spec.Resources.Requests.Storage().String())
	// Depending on the storage class available, readWriteMany may not be supported. Don't wait for the app to deploy successfully because it may not.
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

func TestUsingComputeClasses(t *testing.T) {
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
		name              string
		noComputeClass    bool
		testDataDirectory string
		computeClass      adminv1.ProjectComputeClassInstance
		expected          map[string]v1.Scheduling
		waitFor           func(obj *apiv1.App) bool
		fail              bool
	}{
		{
			name:              "valid",
			testDataDirectory: "./testdata/computeclass",
			computeClass: adminv1.ProjectComputeClassInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "acorn-test-custom",
					Namespace: c.GetNamespace(),
				},
				CPUScaler: 0.25,
				Memory: adminv1.ComputeClassMemory{
					Min: "512Mi",
					Max: "1Gi",
				},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi")},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *apiv1.App) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "unrestricted-default-gets-maximum",
			testDataDirectory: "./testdata/computeclass",
			computeClass: adminv1.ProjectComputeClassInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "acorn-test-custom",
					Namespace: c.GetNamespace(),
				},
				CPUScaler: 0.25,
				Memory: adminv1.ComputeClassMemory{
					Max: "1Gi",
				},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi")},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}}},
			waitFor: func(obj *apiv1.App) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "with-values",
			testDataDirectory: "./testdata/computeclass",
			computeClass: adminv1.ProjectComputeClassInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "acorn-test-custom",
					Namespace: c.GetNamespace(),
				},
				CPUScaler: 0.25,
				Memory: adminv1.ComputeClassMemory{
					Default: "1Gi",
					Values: []string{
						"1Gi",
						"2Gi",
					},
				},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi")},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *apiv1.App) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "default",
			testDataDirectory: "./testdata/simple",
			computeClass: adminv1.ProjectComputeClassInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "acorn-test-custom",
					Namespace: c.GetNamespace(),
				},
				Default:   true,
				CPUScaler: 0.25,
				Memory: adminv1.ComputeClassMemory{
					Default: "512Mi",
					Max:     "1Gi",
					Min:     "512Mi",
				},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi")},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("512Mi"),
						corev1.ResourceCPU:    resource.MustParse("125m"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *apiv1.App) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "does-not-exist",
			noComputeClass:    true,
			testDataDirectory: "./testdata/computeclass",
			fail:              true,
		},
	}

	for _, tt := range checks {
		asClusterComputeClass := adminv1.ClusterComputeClassInstance(tt.computeClass)
		// Perform the same test cases on both Project and Cluster ComputeClasses
		for kind, computeClass := range map[string]crClient.Object{"projectcomputeclass": &tt.computeClass, "clustercomputeclass": &asClusterComputeClass} {
			testcase := fmt.Sprintf("%v-%v", kind, tt.name)
			t.Run(testcase, func(t *testing.T) {
				if !tt.noComputeClass {
					if err := kclient.Create(ctx, computeClass); err != nil {
						t.Fatal(err)
					}

					// Clean-up and gurantee the computeclass doesn't exist after this test run
					t.Cleanup(func() {
						if err = kclient.Delete(context.Background(), computeClass); err != nil && !apierrors.IsNotFound(err) {
							t.Fatal(err)
						}
						err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
							lookingFor := computeClass
							err := kclient.Get(ctx, router.Key(computeClass.GetNamespace(), computeClass.GetName()), lookingFor)
							return lookingFor, err
						})
						if err != nil {
							t.Fatal(err)
						}
					})
				}

				image, err := c.AcornImageBuild(ctx, tt.testDataDirectory+"/Acornfile", &client.AcornImageBuildOptions{
					Cwd: tt.testDataDirectory,
				})
				if err != nil {
					t.Fatal(err)
				}

				// Assign a name for the test case so no collisions occur
				app, err := c.AppRun(ctx, image.ID, &client.AppRunOptions{Name: testcase})
				if err != nil {
					if tt.fail {
						return
					}
					t.Fatal(err)
				}

				// Clean-up and gurantee the app doesn't exist after this test run
				t.Cleanup(func() {
					if err = kclient.Delete(context.Background(), app); err != nil && !apierrors.IsNotFound(err) {
						t.Fatal(err)
					}
					err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
						lookingFor := app
						err := kclient.Get(ctx, router.Key(app.GetName(), app.GetNamespace()), lookingFor)
						return lookingFor, err
					})
					if err != nil {
						t.Fatal(err)
					}
				})

				app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, tt.waitFor)
				assert.EqualValues(t, app.Status.Scheduling, tt.expected, "generated scheduling rules are incorrect")
			})
		}
	}
}

func TestCreatingComputeClasses(t *testing.T) {
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
		memory    adminv1.ComputeClassMemory
		cpuScaler float64
		fail      bool
	}{
		{
			name: "valid-only-max",
			memory: adminv1.ComputeClassMemory{
				Max: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-only-min",
			memory: adminv1.ComputeClassMemory{
				Min: "512Mi",
			},
			fail: false,
		},
		{
			name: "valid-only-default",
			memory: adminv1.ComputeClassMemory{
				Default: "512Mi",
			},
			fail: false,
		},
		{
			name:      "valid-values",
			cpuScaler: 0.25,
			memory: adminv1.ComputeClassMemory{
				Default: "1Gi",
				Values:  []string{"1Gi", "2Gi"},
			},
		},
		{
			name: "valid-empty",
		},
		{
			name: "invalid-memory-default",
			memory: adminv1.ComputeClassMemory{
				Default: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-min",
			memory: adminv1.ComputeClassMemory{
				Min: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-max",
			memory: adminv1.ComputeClassMemory{
				Max: "invalid",
			},
			fail: true,
		},
		{
			name: "invalid-memory-values",
			memory: adminv1.ComputeClassMemory{
				Values: []string{"invalid"},
			},
			fail: true,
		},
		{
			name: "invalid-default-less-than-min",
			memory: adminv1.ComputeClassMemory{
				Default: "128Mi",
				Min:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-default-greater-than-max",
			memory: adminv1.ComputeClassMemory{
				Default: "1Gi",
				Max:     "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-min-max-swapped",
			memory: adminv1.ComputeClassMemory{
				Min: "1Gi",
				Max: "512Mi",
			},
			fail: true,
		},
		{
			name: "invalid-default-for-values",
			memory: adminv1.ComputeClassMemory{
				Default: "128Mi",
				Values:  []string{"512Mi"},
			},
			fail: true,
		},
		{
			name: "invalid-min-max-set-with-values",
			memory: adminv1.ComputeClassMemory{
				Min:    "512Mi",
				Max:    "4Gi",
				Values: []string{"2Gi", "3Gi"},
			},
			fail: true,
		},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			// Create a non-instanced ComputeClass to trigger Mink valdiation
			computeClass := adminapiv1.ProjectComputeClass{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "acorn-test-custom",
					Namespace:    c.GetNamespace(),
				},
				CPUScaler: tt.cpuScaler,
				Memory:    tt.memory,
			}

			// TODO - dry run
			err = kclient.Create(ctx, &computeClass)
			if err != nil && !tt.fail {
				t.Fatal("did not expect creation to fail:", err)
			} else if err == nil {
				if err := kclient.Delete(context.Background(), &computeClass); err != nil && !apierrors.IsNotFound(err) {
					t.Fatal("failed to cleanup test:", err)
				}
				if tt.fail {
					t.Fatal("expected an error to occur when creating an invalid ComputeClass but did not receive one")
				}
			}

		})
	}

}

func TestCrossProjectNetworkConnection(t *testing.T) {
	helper.StartController(t)

	rc, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)
	kc := helper.MustReturn(kclient.Default)

	// create two separate projects in which to run two Nginx apps
	proj1, err := c.ProjectCreate(ctx, "proj1")
	if err != nil {
		t.Fatal("error while creating project:", err)
	}
	proj1Client, err := client.New(rc, proj1.Name, proj1.Status.Namespace)
	if err != nil {
		t.Fatal("error creating client for proj1:", err)
	}

	proj2, err := c.ProjectCreate(ctx, "proj2")
	if err != nil {
		t.Fatal("error while creating project:", err)
	}
	proj2Client, err := client.New(rc, proj2.Name, proj2.Status.Namespace)
	if err != nil {
		t.Fatal("error creating client for proj2:", err)
	}

	t.Cleanup(func() {
		// clean up projects
		_, err := proj1Client.ProjectDelete(ctx, "proj1")
		if err != nil {
			t.Log("failed to delete project 'proj1':", err)
		}
		_, err = proj2Client.ProjectDelete(ctx, "proj2")
		if err != nil {
			t.Log("failed to delete project 'proj2':", err)
		}
	})

	// create both apps, one in proj1 and the other in proj2
	// app "foo" in proj1 does not publish any ports
	// app "bar" in proj2 publishes port 80
	fooImage, err := proj1Client.AcornImageBuild(ctx, "testdata/networkpolicy/Acornfile", nil)
	if err != nil {
		t.Fatal("error while building image:", err)
	}
	fooApp, err := proj1Client.AppRun(ctx, fooImage.ID, &client.AppRunOptions{
		Name:            "foo",
		TargetNamespace: proj1.Namespace,
	})
	if err != nil {
		t.Fatal("error while running app:", err)
	}

	barImage, err := proj2Client.AcornImageBuild(ctx, "testdata/networkpolicy/publish.Acornfile", nil)
	if err != nil {
		t.Fatal("error while building image:", err)
	}
	barApp, err := proj2Client.AppRun(ctx, barImage.ID, &client.AppRunOptions{
		Name:            "bar",
		TargetNamespace: proj2.Namespace,
	})
	if err != nil {
		t.Fatal("error while running app:", err)
	}

	// wait for both apps to be ready
	helper.WaitForObject(t, helper.Watcher(t, proj1Client), &apiv1.AppList{}, fooApp, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionReady).Success
	})
	helper.WaitForObject(t, helper.Watcher(t, proj2Client), &apiv1.AppList{}, barApp, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionReady).Success
	})

	// determine pod IPs so we can test network connections
	fooIP := getPodIPFromAppName(t, ctx, &kc, fooApp.Name, fooApp.Status.Namespace)
	barIP := getPodIPFromAppName(t, ctx, &kc, barApp.Name, barApp.Status.Namespace)

	// build an Acorn that just runs a job with the official curl container
	curlImage1, err := proj1Client.AcornImageBuild(ctx, "testdata/networkpolicy/curl.Acornfile", nil)
	if err != nil {
		t.Fatal("error while building image:", err)
	}
	curlImage2, err := proj2Client.AcornImageBuild(ctx, "testdata/networkpolicy/curl.Acornfile", nil)
	if err != nil {
		t.Fatal("error while building image:", err)
	}

	checks := []struct {
		name          string
		client        client.Client
		podIP         string
		imageID       string
		expectFailure bool
	}{
		{
			name:          "curl-foo-proj1",
			client:        proj1Client,
			podIP:         fooIP,
			imageID:       curlImage1.ID,
			expectFailure: false,
		},
		{
			name:          "curl-bar-proj1",
			client:        proj1Client,
			podIP:         barIP,
			imageID:       curlImage1.ID,
			expectFailure: false,
		},
		{
			name:          "curl-foo-proj2",
			client:        proj2Client,
			podIP:         fooIP,
			imageID:       curlImage2.ID,
			expectFailure: true,
		},
		{
			name:          "curl-bar-proj2",
			client:        proj2Client,
			podIP:         barIP,
			imageID:       curlImage2.ID,
			expectFailure: false,
		},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			// run curl to test the connection
			app, err := check.client.AppRun(ctx, check.imageID, &client.AppRunOptions{
				Name: check.name,
				DeployArgs: map[string]any{
					"address": check.podIP,
				},
			})
			if err != nil {
				t.Fatal("error while running app:", err)
			}

			appInstance := helper.WaitForObject(t, helper.Watcher(t, check.client), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
				return obj.Status.JobsStatus["curl"].Failed || obj.Status.JobsStatus["curl"].Succeed
			})

			if check.expectFailure {
				assert.Equal(t, true, appInstance.Status.JobsStatus["curl"].Failed)
			} else {
				assert.Equal(t, true, appInstance.Status.JobsStatus["curl"].Succeed)
			}
		})
	}
}

func getPodIPFromAppName(t *testing.T, ctx context.Context, kc *runtimeclient.WithWatch, appName, namespace string) string {
	t.Helper()
	var podList corev1.PodList
	selector, err := k8slabels.Parse(fmt.Sprintf("%s=%s", labels.AcornAppName, appName))
	if err != nil {
		t.Fatal("error creating k8s label selector:", err)
	}

	podIP := ""
	for podIP == "" {
		err = (*kc).List(ctx, &podList, &kclient.ListOptions{
			LabelSelector: selector,
			Namespace:     namespace,
		})
		if err != nil {
			t.Fatal("error creating k8s label selector:", err)
		}
		podIP = podList.Items[0].Status.PodIP
	}

	return podIP
}

func getStorageClassName(t *testing.T, storageClasses *storagev1.StorageClassList) string {
	t.Helper()
	if len(storageClasses.Items) == 0 {
		return ""
	}

	storageClassName := storageClasses.Items[0].Name
	// Use local-=path if it exists
	for _, sc := range storageClasses.Items {
		if sc.Name == "local-path" {
			storageClassName = sc.Name
			break
		}
	}
	return storageClassName
}
