package services

import (
	"testing"
	"time"

	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestServices(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/main/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/main",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	k, err := c.GetClient()
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, k.Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppStatus.Jobs["test"].Ready
	})
}

func TestServiceIgnoreCleanup(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/ignorecleanup/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/ignorecleanup",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AppRun(ctx, image.ID, &client.AppRunOptions{Name: "myapp"})
	if err != nil {
		t.Fatal(err)
	}

	// Sleep for 1 second to give the controller time to create the service.
	time.Sleep(1 * time.Second)

	// Delete the app.
	if _, err := c.AppDelete(ctx, "myapp"); err != nil {
		t.Fatal(err)
	}

	// This app's service has a delete event job that will fail to run.
	// Make sure it still shows up in the app list.
	list, err := c.AppList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, list, 1)
	assert.Equal(t, "myapp.myservice", list[0].Name)

	// Delete it next.
	if err := c.AppIgnoreDeleteCleanup(ctx, "myapp.myservice"); err != nil {
		t.Fatal(err)
	}

	// Make sure there are no apps remaining.
	list, err = c.AppList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, list, 0)
}
