package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/integration/helper"
	adminapiv1 "github.com/acorn-io/runtime/pkg/apis/admin.acorn.io/v1"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/appdefinition"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagesource"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/run"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/tolerations"
	"github.com/acorn-io/z"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestVolume(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

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

	pv := helper.Wait(t, kc.Watch, &corev1.PersistentVolumeList{}, func(obj *corev1.PersistentVolume) bool {
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

	helper.WaitForObject(t, kc.Watch, &corev1.PersistentVolumeList{}, pv, func(obj *corev1.PersistentVolume) bool {
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
	c, _ := helper.ClientAndProject(t)

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

func TestServiceConsumer(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)
	kc := helper.MustReturn(kclient.Default)

	image, err := c.AcornImageBuild(ctx, "./testdata/serviceconsumer/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/serviceconsumer/",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Integration tests don't have proper privileges so we will by pass the permission validation
	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-",
			Namespace:    c.GetProject(),
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
			GrantedPermissions: []v1.Permissions{{
				ServiceName: "producer.default",
				Rules: []v1.PolicyRule{{
					PolicyRule: rbacv1.PolicyRule{
						APIGroups: []string{""},
						Verbs:     []string{"get"},
						Resources: []string{"secrets"},
					},
				}},
			}},
		},
	}
	if err := kc.Create(ctx, appInstance); err != nil {
		t.Fatal(err)
	}

	app, err := c.AppGet(ctx, appInstance.Name)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Ready
	})
}

func TestVolumeBadClassInImageBoundToGoodClass(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kc.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping VolumeBadClassInImageBoundToGoodClass")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: getStorageClassName(t, storageClasses),
		SupportedRegions: []string{apiv1.LocalRegion},
	}
	if err = kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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

	helper.Wait(t, kc.Watch, &corev1.PersistentVolumeClaimList{}, func(obj *corev1.PersistentVolumeClaim) bool {
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
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	storageClasses := new(storagev1.StorageClassList)
	if err := kc.List(ctx, storageClasses); err != nil {
		t.Fatal(err)
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: getStorageClassName(t, storageClasses),
		SupportedRegions: []string{apiv1.LocalRegion},
	}

	if err := kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		Inactive:         true,
		SupportedRegions: []string{apiv1.LocalRegion},
	}
	if err := kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Size: adminv1.VolumeClassSize{
			Min: "10Gi",
			Max: "100Gi",
		},
		SupportedRegions: []string{apiv1.LocalRegion},
	}
	if err := kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta: metav1.ObjectMeta{Name: "acorn-test-custom"},
		Size: adminv1.VolumeClassSize{
			Min: "10Gi",
			Max: "100Gi",
		},
		SupportedRegions: []string{apiv1.LocalRegion},
	}
	if err := kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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
	kc := helper.MustReturn(kclient.Default)
	c, _ := helper.ClientAndProject(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kc.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping VolumeClassRemoved")
		return
	}

	volumeClass := adminapiv1.ClusterVolumeClass{
		ObjectMeta:       metav1.ObjectMeta{Name: "acorn-test-custom"},
		StorageClassName: getStorageClassName(t, storageClasses),
		SupportedRegions: []string{apiv1.LocalRegion},
	}
	if err = kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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

	helper.Wait(t, kc.Watch, &corev1.PersistentVolumeClaimList{}, func(obj *corev1.PersistentVolumeClaim) bool {
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
			if err = kc.Delete(ctx, &volumeClass); err != nil {
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
	c, _ := helper.ClientAndProject(t)

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
		SupportedRegions: []string{apiv1.LocalRegion},
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
	c, _ := helper.ClientAndProject(t)

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
		SupportedRegions: []string{apiv1.LocalRegion},
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
	c, _ := helper.ClientAndProject(t)

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
		SupportedRegions: []string{apiv1.LocalRegion},
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
	c, _ := helper.ClientAndProject(t)

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
		Default:          true,
		SupportedRegions: []string{apiv1.LocalRegion},
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
	c, _ := helper.ClientAndProject(t)

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
	c, _ := helper.ClientAndProject(t)

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
		SupportedRegions: []string{apiv1.LocalRegion},
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
	c, _ := helper.ClientAndProject(t)

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
		return mapping["sha256:"+digest] == "ghcr.io/acorn-io/images-mirror/nginx:latest"
	})
}

func TestSimple(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

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
	c, project := helper.ClientAndProject(t)
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

	_, err = appDef.ToParamSpec()
	if err != nil {
		t.Fatal(err)
	}

	appInstance, err := run.Run(helper.GetCTX(t), kclient, &v1.AppInstance{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
			DeployArgs: v1.NewGenericMap(map[string]any{
				"someInt": 5,
			}),
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

func TestRequireComputeClass(t *testing.T) {
	ctx := helper.GetCTX(t)

	helper.StartController(t)
	c, _ := helper.ClientAndProject(t)
	kc := helper.MustReturn(kclient.Default)

	helper.SetRequireComputeClassWithRestore(t, ctx, kc)

	checks := []struct {
		name              string
		noComputeClass    bool
		testDataDirectory string
		computeClass      adminv1.ProjectComputeClassInstance
		expected          map[string]v1.Scheduling
		waitFor           func(obj *v1.AppInstance) bool
		fail              bool
		failMessage       string
	}{
		{
			name:              "no-computeclass",
			noComputeClass:    true,
			testDataDirectory: "./testdata/simple",
			waitFor: func(obj *v1.AppInstance) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Error &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Message == "compute class required but none configured"
			},
		},
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
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"mygpu/nvidia": resource.MustParse("1"),
					}, Requests: corev1.ResourceList{
						"mygpu/nvidia": resource.MustParse("1"),
					}},
				SupportedRegions: []string{apiv1.LocalRegion},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						"mygpu/nvidia":        resource.MustParse("1"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
						"mygpu/nvidia":        resource.MustParse("1"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *v1.AppInstance) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
	}

	for _, tt := range checks {
		asClusterComputeClass := adminv1.ClusterComputeClassInstance(tt.computeClass)
		// Perform the same test cases on both Project and Cluster ComputeClasses
		for kind, computeClass := range map[string]crClient.Object{"projectcomputeclass": &tt.computeClass, "clustercomputeclass": &asClusterComputeClass} {
			testcase := fmt.Sprintf("%v-%v", kind, tt.name)
			t.Run(testcase, func(t *testing.T) {
				if !tt.noComputeClass {
					if err := kc.Create(ctx, computeClass); err != nil {
						t.Fatal(err)
					}

					// Clean-up and gurantee the computeclass doesn't exist after this test run
					t.Cleanup(func() {
						if err := kc.Delete(context.Background(), computeClass); err != nil && !apierrors.IsNotFound(err) {
							t.Fatal(err)
						}
						err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
							lookingFor := computeClass
							err := kc.Get(ctx, router.Key(computeClass.GetNamespace(), computeClass.GetName()), lookingFor)
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
				if err == nil && tt.fail {
					t.Fatal("expected error, got nil")
				} else if err != nil {
					if !tt.fail {
						t.Fatal(err)
					}
					assert.Contains(t, err.Error(), tt.failMessage)
				}

				// Clean-up and gurantee the app doesn't exist after this test run
				if app != nil {
					t.Cleanup(func() {
						if err = kc.Delete(context.Background(), app); err != nil && !apierrors.IsNotFound(err) {
							t.Fatal(err)
						}
						err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
							lookingFor := app
							err := kc.Get(ctx, router.Key(app.GetName(), app.GetNamespace()), lookingFor)
							return lookingFor, err
						})
						if err != nil {
							t.Fatal(err)
						}
					})
				}

				if tt.waitFor != nil {
					appInstance := &v1.AppInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      app.Name,
							Namespace: app.Namespace,
						},
					}
					appInstance = helper.WaitForObject(t, kc.Watch, new(v1.AppInstanceList), appInstance, tt.waitFor)
					assert.EqualValues(t, appInstance.Status.Scheduling, tt.expected, "generated scheduling rules are incorrect")
				}
			})
		}
	}
}

func TestUsingComputeClasses(t *testing.T) {
	helper.StartController(t)
	c, _ := helper.ClientAndProject(t)
	kc := helper.MustReturn(kclient.Default)

	ctx := helper.GetCTX(t)

	checks := []struct {
		name              string
		noComputeClass    bool
		testDataDirectory string
		computeClass      adminv1.ProjectComputeClassInstance
		expected          map[string]v1.Scheduling
		waitFor           func(obj *v1.AppInstance) bool
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
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						"mygpu/nvidia": resource.MustParse("1"),
					}, Requests: corev1.ResourceList{
						"mygpu/nvidia": resource.MustParse("1"),
					}},
				SupportedRegions: []string{apiv1.LocalRegion},
			},
			expected: map[string]v1.Scheduling{"simple": {
				Requirements: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						"mygpu/nvidia":        resource.MustParse("1"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("250m"),
						"mygpu/nvidia":        resource.MustParse("1"),
					},
				},
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *v1.AppInstance) bool {
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
				SupportedRegions: []string{apiv1.LocalRegion},
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
			waitFor: func(obj *v1.AppInstance) bool {
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
				SupportedRegions: []string{apiv1.LocalRegion},
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
			waitFor: func(obj *v1.AppInstance) bool {
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
				SupportedRegions: []string{apiv1.LocalRegion},
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
			waitFor: func(obj *v1.AppInstance) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "priority-class",
			testDataDirectory: "./testdata/simple",
			computeClass: adminv1.ProjectComputeClassInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "acorn-test-custom",
					Namespace: c.GetNamespace(),
				},
				Default:           true,
				CPUScaler:         0.25,
				PriorityClassName: "system-cluster-critical",
				SupportedRegions:  []string{apiv1.LocalRegion},
			},
			expected: map[string]v1.Scheduling{"simple": {
				PriorityClassName: "system-cluster-critical",
				Tolerations: []corev1.Toleration{
					{
						Key:      tolerations.WorkloadTolerationKey,
						Operator: corev1.TolerationOpExists,
					},
				}},
			},
			waitFor: func(obj *v1.AppInstance) bool {
				return obj.Status.Condition(v1.AppInstanceConditionParsed).Success &&
					obj.Status.Condition(v1.AppInstanceConditionScheduling).Success
			},
		},
		{
			name:              "unsupported-region",
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
				SupportedRegions: []string{"non-local"},
			},
			fail: true,
		},
		{
			name:              "no-region",
			testDataDirectory: "./testdata/computeclass",
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
			fail: true,
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
					if err := kc.Create(ctx, computeClass); err != nil {
						t.Fatal(err)
					}

					// Clean-up and gurantee the computeclass doesn't exist after this test run
					t.Cleanup(func() {
						if err := kc.Delete(context.Background(), computeClass); err != nil && !apierrors.IsNotFound(err) {
							t.Fatal(err)
						}
						err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
							lookingFor := computeClass
							err := kc.Get(ctx, router.Key(computeClass.GetNamespace(), computeClass.GetName()), lookingFor)
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
				if err == nil && tt.fail {
					t.Fatal("expected error, got nil")
				} else if err != nil {
					if !tt.fail {
						t.Fatal(err)
					}
				}

				// Clean-up and gurantee the app doesn't exist after this test run
				if app != nil {
					t.Cleanup(func() {
						if err = kc.Delete(context.Background(), app); err != nil && !apierrors.IsNotFound(err) {
							t.Fatal(err)
						}
						err := helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
							lookingFor := app
							err := kc.Get(ctx, router.Key(app.GetName(), app.GetNamespace()), lookingFor)
							return lookingFor, err
						})
						if err != nil {
							t.Fatal(err)
						}
					})
				}

				if tt.waitFor != nil {
					appInstance := &v1.AppInstance{
						ObjectMeta: metav1.ObjectMeta{
							Name:      app.Name,
							Namespace: app.Namespace,
						},
					}
					appInstance = helper.WaitForObject(t, kc.Watch, new(v1.AppInstanceList), appInstance, tt.waitFor)
					assert.EqualValues(t, appInstance.Status.Scheduling, tt.expected, "generated scheduling rules are incorrect")
				}
			})
		}
	}
}

func TestJobDelete(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/jobfinalize/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/jobfinalize",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(app *apiv1.App) bool {
		return len(app.Finalizers) > 0
	})

	app, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
		return c.AppGet(ctx, app.Name)
	})
}

func TestAppWithBadRegion(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/simple/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{Region: "does-not-exist"})
	if err == nil || !strings.Contains(err.Error(), "is not supported for project") {
		t.Fatalf("expected an invalid region error, got %v", err)
	}
}

func TestAppWithBadDefaultRegion(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	kc := helper.MustReturn(kclient.Default)
	c, project := helper.ClientAndProject(t)

	storageClasses := new(storagev1.StorageClassList)
	err := kc.List(ctx, storageClasses)
	if err != nil || len(storageClasses.Items) == 0 {
		t.Skip("No storage classes, so skipping TestAppWithBadDefaultRegion")
		return
	}

	volumeClass := adminapiv1.ProjectVolumeClass{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: project.Name,
			Name:      "acorn-test-custom",
		},
		StorageClassName: getStorageClassName(t, storageClasses),
		Default:          true,
		SupportedRegions: []string{"custom"},
	}
	if err = kc.Create(ctx, &volumeClass); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = kc.Delete(context.Background(), &volumeClass); err != nil && !apierrors.IsNotFound(err) {
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
	if err == nil || !strings.Contains(err.Error(), "is not a valid volume class") {
		t.Fatalf("expected an invalid region error, got %v", err)
	}
}

func getStorageClassName(t *testing.T, storageClasses *storagev1.StorageClassList) string {
	t.Helper()
	if len(storageClasses.Items) == 0 {
		return ""
	}

	storageClassName := storageClasses.Items[0].Name
	// Use local-path if it exists
	for _, sc := range storageClasses.Items {
		if sc.Name == "local-path" {
			storageClassName = sc.Name
			break
		}
	}
	return storageClassName
}

func TestCrossProjectNetworkConnection(t *testing.T) {
	helper.StartController(t)

	rc, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)
	kc := helper.MustReturn(kclient.Default)

	cfg, err := config.Get(ctx, kc)
	if err != nil {
		t.Fatal(err)
	} else if !*cfg.NetworkPolicies {
		t.SkipNow() // skip this test because NetworkPolicies are not enabled
	}

	// create two separate projects in which to run two Nginx apps
	proj1, err := c.ProjectCreate(ctx, "proj1", apiv1.LocalRegion, []string{apiv1.LocalRegion})
	if err != nil {
		t.Fatal("error while creating project:", err)
	}
	proj1Client, err := client.New(rc, proj1.Name, proj1.Name)
	if err != nil {
		t.Fatal("error creating client for proj1:", err)
	}

	proj2, err := c.ProjectCreate(ctx, "proj2", apiv1.LocalRegion, []string{apiv1.LocalRegion})
	if err != nil {
		t.Fatal("error while creating project:", err)
	}
	proj2Client, err := client.New(rc, proj2.Name, proj2.Name)
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
	fooApp, err := proj1Client.AppRun(ctx, fooImage.ID, &client.AppRunOptions{Name: "foo"})
	if err != nil {
		t.Fatal("error while running app:", err)
	}

	barImage, err := proj2Client.AcornImageBuild(ctx, "testdata/networkpolicy/publish.Acornfile", nil)
	if err != nil {
		t.Fatal("error while building image:", err)
	}
	barApp, err := proj2Client.AppRun(ctx, barImage.ID, &client.AppRunOptions{Name: "bar"})
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
	fooIP := getPodIPFromAppName(ctx, t, &kc, fooApp.Name, fooApp.Status.Namespace)
	barIP := getPodIPFromAppName(ctx, t, &kc, barApp.Name, barApp.Status.Namespace)

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
			name:    "curl-foo-proj1",
			client:  proj1Client,
			podIP:   fooIP,
			imageID: curlImage1.ID,
		},
		{
			name:    "curl-bar-proj1",
			client:  proj1Client,
			podIP:   barIP,
			imageID: curlImage1.ID,
			// even though bar has port 80 published, it should not be reachable from anything outside
			// proj2 except for the ingress controller, so failure is expected here
			expectFailure: true,
		},
		{
			name:          "curl-foo-proj2",
			client:        proj2Client,
			podIP:         fooIP,
			imageID:       curlImage2.ID,
			expectFailure: true,
		},
		{
			name:    "curl-bar-proj2",
			client:  proj2Client,
			podIP:   barIP,
			imageID: curlImage2.ID,
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
				return obj.Status.AppStatus.Jobs["curl"].Ready || obj.Status.AppStatus.Jobs["curl"].ErrorCount > 0
			})

			if check.expectFailure {
				assert.Equal(t, true, appInstance.Status.AppStatus.Jobs["curl"].ErrorCount > 0)
			} else {
				assert.Equal(t, true, appInstance.Status.AppStatus.Jobs["curl"].Ready)
			}
		})
	}
}

func getPodIPFromAppName(ctx context.Context, t *testing.T, kc *crClient.WithWatch, appName, namespace string) string {
	t.Helper()
	selector, err := k8slabels.Parse(fmt.Sprintf("%s=%s", labels.AcornAppName, appName))
	if err != nil {
		t.Fatal("error creating k8s label selector:", err)
	}

	var podList corev1.PodList
	podIP := ""
	for podIP == "" {
		err = (*kc).List(ctx, &podList, &kclient.ListOptions{
			LabelSelector: selector,
			Namespace:     namespace,
		})
		if err != nil {
			t.Fatal("error listing pods:", err)
		}
		podIP = podList.Items[0].Status.PodIP
	}

	return podIP
}

func TestProjectUpdate(t *testing.T) {
	helper.StartController(t)

	rc, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)
	projectName := uuid.New().String()[:8]
	proj1, err := c.ProjectCreate(ctx, projectName, apiv1.LocalRegion, []string{apiv1.LocalRegion})
	if err != nil {
		t.Fatal("error while creating project:", err)
	}
	proj1Client, err := client.New(rc, proj1.Name, proj1.Status.Namespace)
	if err != nil {
		t.Fatal("error creating client for project:", err)
	}
	t.Cleanup(func() {
		// clean up projects
		_, err := proj1Client.ProjectDelete(ctx, projectName)
		if err != nil {
			t.Logf("failed to delete project '%s': %s", projectName, err)
		}
	})

	var updatedProj *apiv1.Project
	// update default
	for i := 0; i < 10; i++ {
		updatedProj, err = proj1Client.ProjectUpdate(ctx, proj1, "new-default", []string{apiv1.LocalRegion})
		if err == nil {
			break
		}
		if !apierrors.IsConflict(err) {
			t.Fatal("error while updating project:", err)
		}
		proj1, err = proj1Client.ProjectGet(ctx, projectName)
		if err != nil {
			t.Fatal("error while getting project:", err)
		}
	}

	assert.Equal(t, updatedProj.Spec.DefaultRegion, "new-default")
	assert.Equal(t, updatedProj.Spec.SupportedRegions, []string{apiv1.LocalRegion, "new-default"})

	// swap default from new-default to local
	for i := 0; i < 10; i++ {
		updatedProj, err = proj1Client.ProjectGet(ctx, projectName)
		if err != nil {
			t.Fatal("error while getting project:", err)
		}
		updatedProj, err = proj1Client.ProjectUpdate(ctx, updatedProj, apiv1.LocalRegion, nil)
		if err == nil {
			break
		}
		if !apierrors.IsConflict(err) {
			t.Fatal("error while updating project:", err)
		}
	}
	assert.Equal(t, updatedProj.Spec.DefaultRegion, apiv1.LocalRegion)
	assert.Equal(t, updatedProj.Spec.SupportedRegions, []string{apiv1.LocalRegion, "new-default"})

	// remove new-default region
	for i := 0; i < 10; i++ {
		updatedProj, err = proj1Client.ProjectGet(ctx, projectName)
		if err != nil {
			t.Fatal("error while getting project:", err)
		}
		updatedProj, err = proj1Client.ProjectUpdate(ctx, updatedProj, "", []string{apiv1.LocalRegion})
		if err == nil {
			break
		}
		if !strings.Contains(err.Error(), "please apply your changes to the latest version and try again") {
			t.Fatal("error while updating project:", err)
		}
	}
	assert.Equal(t, updatedProj.Spec.DefaultRegion, apiv1.LocalRegion)
	assert.Equal(t, updatedProj.Spec.SupportedRegions, []string{apiv1.LocalRegion})

	// set supported regions
	for i := 0; i < 10; i++ {
		updatedProj, err = proj1Client.ProjectGet(ctx, projectName)
		if err != nil {
			t.Fatal("error while getting project:", err)
		}
		updatedProj, err = proj1Client.ProjectUpdate(ctx, updatedProj, "", []string{apiv1.LocalRegion, "local3", "local2"})
		if err == nil {
			break
		}
		if !apierrors.IsConflict(err) {
			t.Fatal("error while updating project:", err)
		}
	}
	assert.Equal(t, updatedProj.Spec.DefaultRegion, apiv1.LocalRegion)
	assert.Equal(t, updatedProj.Spec.SupportedRegions, []string{apiv1.LocalRegion, "local3", "local2"})
}

func TestEnforcedQuota(t *testing.T) {
	ctx := helper.GetCTX(t)

	helper.StartController(t)
	restConfig, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	// Create a project.
	kc := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kc)

	// Create a client for the project.
	c, err := client.New(restConfig, project.Name, project.Name)
	if err != nil {
		t.Fatal(err)
	}

	// Annotate the project to enforce quota.
	helper.WaitForObject(t, helper.Watcher(t, c), &v1.ProjectInstanceList{}, project, func(obj *v1.ProjectInstance) bool {
		if obj.Annotations == nil {
			obj.Annotations = make(map[string]string)
		}
		obj.Annotations[labels.ProjectEnforcedQuotaAnnotation] = "true"
		return kc.Update(ctx, obj) == nil
	})

	// Run a scaled app.
	image, err := c.AcornImageBuild(ctx, "./testdata/scaled/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/scaled",
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err = c.AppDelete(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
	})

	// Wait for the app to set the AppInstanceQuota condition to be transitioning and for the namespace
	// to be ready.
	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionQuota).Transitioning
	})

	// Grab the app's QuotaRequest and check that it has the appropriate values set.
	quotaRequest := &adminv1.QuotaRequestInstance{}
	err = kc.Get(ctx, router.Key(app.Namespace, app.Name), quotaRequest)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 2, quotaRequest.Spec.Resources.Containers)

	// Update the status of the QuotaRequest to communicate readiness.
	quotaRequest.Status = adminv1.QuotaRequestInstanceStatus{
		ObservedGeneration: quotaRequest.Generation,
		Conditions: []v1.Condition{{
			ObservedGeneration: quotaRequest.Generation,
			Type:               adminv1.QuotaRequestCondition,
			Status:             metav1.ConditionTrue,
			Success:            true,
		}},
		AllocatedResources: quotaRequest.Spec.Resources,
	}
	err = kc.Status().Update(ctx, quotaRequest)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the app to set the AppInstanceQuota condition to success.
	helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionQuota).Success
	})
}

func TestAutoUpgradeImageValidation(t *testing.T) {
	ctx := helper.GetCTX(t)

	helper.StartController(t)
	restConfig, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	kc := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kc)

	c, err := client.New(restConfig, project.Name, project.Name)
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, "ghcr.io/acorn-io/library/nginx:latest", &client.AppRunOptions{
		Name:        "myapp",
		AutoUpgrade: z.Pointer(true),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to update the app to "myimage:latest".
	// Since no image exists with this tag, it should fail.
	// Auto-upgrade apps are not supposed to implicitly use Docker Hub when no registry is specified.
	_, err = c.AppUpdate(ctx, app.Name, &client.AppUpdateOptions{
		Image: "myimage:latest",
	})
	if err == nil {
		t.Fatal("expected error when failing to find local image for auto-upgrade app, got no error")
	}
	assert.ErrorContains(t, err, "could not find local image for myimage:latest - if you are trying to use a remote image, specify the full registry")
}

func TestAutoUpgradeLocalImage(t *testing.T) {
	ctx := helper.GetCTX(t)

	helper.StartController(t)
	restConfig, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	kc := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kc)

	c, err := client.New(restConfig, project.Name, project.Name)
	if err != nil {
		t.Fatal(err)
	}

	// Attempt to run an auto-upgrade app with a non-existent local image. Should get an error.
	_, err = c.AppRun(ctx, "mylocalimage", &client.AppRunOptions{
		AutoUpgrade: z.Pointer(true),
	})
	if err == nil {
		t.Fatalf("expected to get a not found error, instead got %v", err)
	}
	assert.ErrorContains(t, err, "could not find local image for mylocalimage - if you are trying to use a remote image, specify the full registry")

	// Next, build the local image
	image, err := c.AcornImageBuild(ctx, "./testdata/named/Acornfile", &client.AcornImageBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Tag the image
	err = c.ImageTag(ctx, image.ID, "mylocalimage")
	if err != nil {
		t.Fatal(err)
	}

	// Deploy the app
	imageSource := imagesource.NewImageSource("", "", "", []string{"mylocalimage"}, nil, true)
	appImage, _, _, err := imageSource.GetImageAndDeployArgs(ctx, c)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, appImage, &client.AppRunOptions{
		AutoUpgrade: z.Pointer(true),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestIgnoreResourceRequirements(t *testing.T) {
	ctx := helper.GetCTX(t)

	helper.StartController(t)
	restConfig, err := restconfig.New(scheme.Scheme)
	if err != nil {
		t.Fatal("error while getting rest config:", err)
	}
	kc := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kc)

	helper.SetIgnoreResourceRequirementsWithRestore(ctx, t, kc)

	c, err := client.New(restConfig, project.Name, project.Name)
	if err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(ctx, "./testdata/simple/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	// deploy an app with memory request configured, verify that it doesn't have the constraints
	app, err := c.AppRun(ctx, image.ID, &client.AppRunOptions{
		Memory: map[string]*int64{
			"": z.Pointer(int64(1073741824)),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
	}
	appInstance = helper.WaitForObject(t, helper.Watcher(t, c), &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})
	assert.Empty(t, appInstance.Status.Scheduling["simple"].Requirements)
}
