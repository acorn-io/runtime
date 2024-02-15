package run

import (
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	crClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestJobDelete(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/jobs/finalize/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/jobs/finalize",
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

func TestCronJobWithCreate(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/jobs/cron-with-create/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/jobs/cron-with-create",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(app *apiv1.App) bool {
		return app.Status.Ready
	})

	app, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
		return c.AppGet(ctx, app.Name)
	})
}

func TestCronJobWithUpdate(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/jobs/cron-with-update/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/jobs/cron-with-update",
	})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(app *apiv1.App) bool {
		return app.Status.Ready && app.Status.AppStatus.Jobs["update"].Skipped
	})

	app, err = c.AppUpdate(ctx, app.Name, &client.AppUpdateOptions{
		DeployArgs: map[string]any{
			"forceUpdateGen": 2,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), new(apiv1.AppList), app, func(app *apiv1.App) bool {
		return app.Status.Ready && !app.Status.AppStatus.Jobs["update"].Skipped
	})

	app, err = c.AppDelete(ctx, app.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = helper.EnsureDoesNotExist(ctx, func() (crClient.Object, error) {
		return c.AppGet(ctx, app.Name)
	})
}
