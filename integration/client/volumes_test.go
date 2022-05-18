package client

import (
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestVolumeListGetDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := newImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Namespace != ""
	})

	helper.Wait(t, kclient.Watch, &corev1.PersistentVolumeList{}, func(pv *corev1.PersistentVolume) bool {
		return pv.Labels[labels.AcornAppName] == app.Name &&
			pv.Labels[labels.AcornAppNamespace] == app.Namespace
	})

	vols, err := c.VolumeList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, vols, 1)
	assert.Equal(t, "10G", vols[0].Spec.Capacity.String())
	assert.Equal(t, "local-path", vols[0].Spec.Class)
	assert.Equal(t, ns.Name, vols[0].Namespace)

	vol, err := c.VolumeGet(ctx, vols[0].Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, vols[0].UID, vol.UID)
	assert.Equal(t, ns.Name, vol.Namespace)

	delVol, err := c.VolumeDelete(ctx, vol.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, delVol)
	assert.Equal(t, vol.UID, delVol.UID)

	_, err = c.VolumeDelete(ctx, vol.Name)
	if err != nil {
		t.Fatal(err)
	}
}

func TestVolumeWatch(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := newImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.Wait(t, lclient.Watch, &apiv1.VolumeList{}, func(vol *apiv1.Volume) bool {
		return vol.Labels[labels.AcornAppName] == app.Name &&
			vol.Labels[labels.AcornAppNamespace] == app.Namespace &&
			vol.Labels[labels.AcornVolumeName] == "vol" &&
			vol.Status.Status == "bound"
	})
}
