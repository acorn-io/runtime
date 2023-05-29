package containers

import (
	"io"
	"strings"
	"testing"

	client2 "github.com/acorn-io/acorn/integration/client"
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Containers["default"].UpToDateReplicaCount == 1
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Namespace != "" && app.Status.AppStatus.Containers["default"].UpToDateReplicaCount == 1
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Containers["default"].UpToDateReplicaCount == 1
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Containers["default"].UpToDateReplicaCount > 0
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)

	con := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[0], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})

	cio, err := c.ContainerReplicaExec(ctx, con.Name, []string{"echo", "test"}, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(cio.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "test", strings.TrimSpace(string(data)))

	exit := <-cio.ExitCode
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.Namespace != "" && app.Status.AppStatus.Containers["default"].UpToDateReplicaCount > 0
	})

	cons, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 1)

	con := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[0], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})

	cio, err := c.ContainerReplicaExec(ctx, con.Name, []string{"cat", "/etc/os-release"}, false, &client.ContainerReplicaExecOptions{
		DebugImage: "ubuntu",
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := io.ReadAll(cio.Stdout)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(strings.ToLower(string(data)), "ubuntu"))

	exit := <-cio.ExitCode
	assert.Equal(t, 0, exit.Code)
}

func TestContainerWithSidecarExec(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	lclient, err := kclient.New(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImageWithSidecar(t, ns.Name)
	app, err := c.AppRun(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, lclient.Watch, &apiv1.AppList{}, app, func(app *apiv1.App) bool {
		return app.Status.AppStatus.Containers["web"].UpToDateReplicaCount > 0
	})

	cons, err := c.ContainerReplicaList(ctx, &client.ContainerReplicaListOptions{App: app.Name})
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, cons, 2)

	webCont := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[0], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})
	sidecarCont := helper.WaitForObject(t, lclient.Watch, &apiv1.ContainerReplicaList{}, &cons[1], func(con *apiv1.ContainerReplica) bool {
		return con.Status.Phase == corev1.PodRunning
	})

	webIO, err := c.ContainerReplicaExec(ctx, webCont.Name, []string{"cat", "/tmp/file"}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	webData, err := io.ReadAll(webIO.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	webExit := <-webIO.ExitCode

	assert.Equal(t, "This is the web container", string(webData))
	assert.Equal(t, 0, webExit.Code)

	sidecarIO, err := c.ContainerReplicaExec(ctx, sidecarCont.Name, []string{"cat", "/tmp/file"}, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	sidecarData, err := io.ReadAll(sidecarIO.Stdout)
	sidecarExit := <-sidecarIO.ExitCode
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "This is the sidecar", string(sidecarData))
	assert.Equal(t, 0, sidecarExit.Code)
}
