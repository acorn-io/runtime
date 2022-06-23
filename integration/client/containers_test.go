package client

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestContainerList(t *testing.T) {
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
		return app.Status.ContainerStatus["default"].UpToDate == 1
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)
	assert.Truef(t, strings.HasPrefix(cons[0].Name, app.Name+"."), "not prefix %s %s", cons[0].Name, app.Name)
	assert.Equal(t, app.Namespace, cons[0].Namespace)
}

func TestContainerDelete(t *testing.T) {
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

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Namespace != "" && app.Status.ContainerStatus["default"].UpToDate == 1
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, "", cons[0].Status.PodName)
	assert.NotEqual(t, "", cons[0].Status.PodNamespace)

	con, err := c.ContainerReplicaDelete(ctx, cons[0].Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, con)

	_, err = c.ContainerReplicaDelete(ctx, cons[0].Name)
	if err != nil {
		t.Fatal(err)
	}
}

func TestContainerGet(t *testing.T) {
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

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["default"].UpToDate == 1
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)

	con, err := c.ContainerReplicaGet(ctx, cons[0].Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cons[0].Name, con.Name)
	assert.Equal(t, cons[0].Namespace, con.Namespace)
	assert.Equal(t, cons[0].UID, con.UID)
}

func TestContainerExec(t *testing.T) {
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

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.ContainerStatus["default"].UpToDate > 0
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)

	con := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[0], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})

	io, err := c.ContainerReplicaExec(ctx, con.Name, []string{"echo", "test"}, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(io.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "test", strings.TrimSpace(string(data)))

	exit := <-io.ExitCode
	assert.Equal(t, 0, exit.Code)
}

func TestContainerDebugExec(t *testing.T) {
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

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Namespace != "" && app.Status.ContainerStatus["default"].UpToDate > 0
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)

	con := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[0], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})

	io, err := c.ContainerReplicaExec(ctx, con.Name, []string{"cat", "/etc/os-release"}, false, &client.ContainerReplicaExecOptions{
		DebugImage: "ubuntu",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(io.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(strings.ToLower(string(data)), "ubuntu"))

	exit := <-io.ExitCode
	assert.Equal(t, 0, exit.Code)
}
