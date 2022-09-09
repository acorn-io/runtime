package client

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/sets"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

func nestedImage(t *testing.T, c client.Client) string {
	t.Helper()

	image, err := build.Build(helper.GetCTX(t), "./testdata/nested/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/nested",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func TestNestApp(t *testing.T) {
	ctx := context.Background()
	c, _ := helper.ClientAndNamespace(t)
	image := nestedImage(t, c)

	rootApp, err := c.AppRun(ctx, image, nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = helper.Wait(t, c.GetClient().Watch, &v1.AppList{}, func(app *v1.App) bool {
		return app.Status.ContainerStatus["level3"].UpToDate == 1 &&
			app.Labels[labels.AcornRootNamespace] == rootApp.Namespace
	})

	_ = helper.Wait(t, c.GetClient().Watch, &v1.AppList{}, func(app *v1.App) bool {
		return app.Status.ContainerStatus["level2"].UpToDate == 1 &&
			app.Labels[labels.AcornRootNamespace] == rootApp.Namespace
	})

	_ = helper.Wait(t, c.GetClient().Watch, &v1.AppList{}, func(app *v1.App) bool {
		return app.Status.ContainerStatus["level1"].UpToDate == 1 &&
			app.Labels[labels.AcornRootNamespace] == rootApp.Namespace
	})

	apps, err := c.AppList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(apps, func(i, j int) bool {
		return apps[i].Name < apps[j].Name
	})

	assert.Len(t, apps, 3)
	assert.Equal(t, rootApp.Name, apps[0].Name)
	assert.Equal(t, rootApp.Namespace, apps[0].Namespace)

	assert.Equal(t, rootApp.Name+".level2", apps[1].Name)
	assert.Equal(t, rootApp.Namespace, apps[1].Namespace)

	assert.Equal(t, rootApp.Name+".level2.level3", apps[2].Name)
	assert.Equal(t, rootApp.Namespace, apps[2].Namespace)

	for _, app := range apps {
		newApp, err := c.AppGet(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, app.Name, newApp.Name)
		assert.Equal(t, app.Namespace, newApp.Namespace)
		assert.Equal(t, app.UID, newApp.UID)
	}

	for _, app := range apps {
		assert.Nil(t, app.Spec.DevMode)
		newApp, err := c.AppUpdate(ctx, app.Name, &client.AppUpdateOptions{
			DevMode: new(bool),
		})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, app.Name, newApp.Name)
		assert.Equal(t, app.Namespace, newApp.Namespace)
		assert.Equal(t, app.UID, newApp.UID)
		assert.Equal(t, false, *newApp.Spec.DevMode)
	}

	w, err := c.GetClient().Watch(ctx, &v1.AppList{}, &client2.ListOptions{Namespace: rootApp.Namespace})
	if err != nil {
		t.Fatal(err)
	}

	names := sets.NewString()
	for event := range w.ResultChan() {
		m, err := meta.Accessor(event.Object)
		if err != nil {
			continue
		}
		assert.Equal(t, rootApp.Namespace, m.GetNamespace())
		names.Insert(m.GetName())
		if names.Len() == 3 {
			break
		}
	}

	w.Stop()
	go func() {
		for range w.ResultChan() {
		}
	}()

	assert.Equal(t, []string{
		rootApp.Name,
		rootApp.Name + ".level2",
		rootApp.Name + ".level2.level3",
	}, names.List())

	for i := 2; i >= 0; i-- {
		app := apps[i]
		newApp, err := c.AppDelete(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, app.Name, newApp.Name)
		assert.Equal(t, app.Namespace, newApp.Namespace)
		assert.Equal(t, app.UID, newApp.UID)
	}
}

func TestNestContainer(t *testing.T) {
	ctx := context.Background()
	c, ns := helper.ClientAndNamespace(t)
	image := nestedImage(t, c)

	app, err := c.AppRun(ctx, image, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, suffix := range []string{"1", "2", "3"} {
		helper.Wait(t, c.GetClient().Watch, &v1.AppList{}, func(app *v1.App) bool {
			return app.Status.ContainerStatus["level"+suffix].UpToDate == 1 &&
				app.Labels[labels.AcornRootNamespace] == ns.Name
		})
	}

	containers, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(containers, func(i, j int) bool {
		return containers[i].Name < containers[j].Name
	})

	assert.Len(t, containers, 6)
	assert.True(t, strings.HasPrefix(containers[0].Name, app.Name+".level1-"))
	assert.Equal(t, app.Namespace, containers[0].Namespace)

	assert.Regexp(t, app.Name+"\\.level1-[a-f0-9]{5,10}-[a-z0-9]{5}\\.side1", containers[1].Name)
	assert.Equal(t, app.Namespace, containers[1].Namespace)

	assert.True(t, strings.HasPrefix(containers[2].Name, app.Name+".level2.level2-"))
	assert.Equal(t, app.Namespace, containers[2].Namespace)

	assert.Regexp(t, app.Name+"\\.level2\\.level2-[a-f0-9]{5,10}-[a-z0-9]{5}\\.side2", containers[3].Name)
	assert.Equal(t, app.Namespace, containers[3].Namespace)

	assert.True(t, strings.HasPrefix(containers[4].Name, app.Name+".level2.level3.level3-"))
	assert.Equal(t, app.Namespace, containers[4].Namespace)

	assert.Regexp(t, app.Name+"\\.level2\\.level3\\.level3-[a-f0-9]{5,10}-[a-z0-9]{5}\\.side3", containers[5].Name)
	assert.Equal(t, app.Namespace, containers[5].Namespace)

	for _, app := range containers {
		newContainer, err := c.ContainerReplicaGet(ctx, app.Name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, app.Name, newContainer.Name)
		assert.Equal(t, app.Namespace, newContainer.Namespace)
		assert.Equal(t, app.UID, newContainer.UID)
	}

	w, err := c.GetClient().Watch(ctx, &v1.ContainerReplicaList{}, &client2.ListOptions{Namespace: app.Namespace})
	if err != nil {
		t.Fatal(err)
	}

	names := sets.NewString()
	for event := range w.ResultChan() {
		m, err := meta.Accessor(event.Object)
		if err != nil {
			continue
		}
		assert.Equal(t, app.Namespace, m.GetNamespace())
		names.Insert(m.GetName())
		if names.Len() == 6 {
			break
		}
	}

	w.Stop()
	go func() {
		for range w.ResultChan() {
		}
	}()

	nameList := names.List()
	assert.Len(t, nameList, 6)
	assert.True(t, strings.HasPrefix(nameList[0], app.Name+".level1-"))
	assert.Regexp(t, app.Name+"\\.level1-[a-f0-9]{8,10}-[a-z0-9]{5}\\.side1", nameList[1])
	assert.True(t, strings.HasPrefix(nameList[2], app.Name+".level2.level2-"))
	assert.Regexp(t, app.Name+"\\.level2\\.level2-[a-f0-9]{8,10}-[a-z0-9]{5}\\.side2", nameList[3])
	assert.True(t, strings.HasPrefix(nameList[4], app.Name+".level2.level3.level3-"))
	assert.Regexp(t, app.Name+"\\.level2\\.level3\\.level3-[a-f0-9]{8,10}-[a-z0-9]{5}\\.side3", nameList[5])

	for i := 5; i >= 0; i-- {
		container := containers[i]
		containerApp, err := c.ContainerReplicaDelete(ctx, container.Name)
		if err != nil {
			t.Fatal(err)
		}
		// two containers are from the same pod, so only one will return something
		if i%2 == 1 {
			assert.Equal(t, container.Name, containerApp.Name)
			assert.Equal(t, container.Namespace, containerApp.Namespace)
			assert.Equal(t, container.UID, containerApp.UID)
		}
	}
}

func TestNestVolume(t *testing.T) {
	ctx := context.Background()
	c, _ := helper.ClientAndNamespace(t)
	kclient := helper.MustReturn(kclient.Default)
	image := nestedImage(t, c)

	app, err := c.AppRun(ctx, image, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, suffix := range []string{".level2", ".level2.level3", ""} {
		helper.Wait(t, kclient.Watch, &corev1.PersistentVolumeList{}, func(pv *corev1.PersistentVolume) bool {
			return pv.Labels[labels.AcornRootPrefix] == app.Name+suffix
		})
	}

	volumes, err := c.VolumeList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(volumes, func(i, j int) bool {
		return volumes[i].Name < volumes[j].Name
	})

	assert.Len(t, volumes, 3)
	assert.Equal(t, app.Namespace, volumes[0].Namespace)
	assert.Equal(t, app.Namespace, volumes[1].Namespace)
	assert.Equal(t, app.Namespace, volumes[2].Namespace)

	for _, vol := range volumes {
		newVol, err := c.VolumeGet(ctx, vol.Name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, vol.Name, newVol.Name)
		assert.Equal(t, vol.Namespace, newVol.Namespace)
		assert.Equal(t, vol.UID, newVol.UID)
	}

	w, err := c.GetClient().Watch(ctx, &v1.VolumeList{}, &client2.ListOptions{Namespace: app.Namespace})
	if err != nil {
		t.Fatal(err)
	}

	names := sets.NewString()
	for event := range w.ResultChan() {
		m, err := meta.Accessor(event.Object)
		if err != nil {
			continue
		}
		assert.Equal(t, app.Namespace, m.GetNamespace())
		names.Insert(m.GetName())
		if names.Len() == 3 {
			break
		}
	}

	w.Stop()
	go func() {
		for range w.ResultChan() {
		}
	}()

	for i := 2; i >= 0; i-- {
		vol := volumes[i]
		deletedVol, err := c.VolumeDelete(ctx, vol.Name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, vol.Name, deletedVol.Name)
		assert.Equal(t, vol.Namespace, deletedVol.Namespace)
		assert.Equal(t, vol.UID, deletedVol.UID)
	}
}
