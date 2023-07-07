package apps

import (
	"testing"

	client2 "github.com/acorn-io/runtime/integration/client"
	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePull(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, project.Name)
	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, "foo", &client.AppRunOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppImage.ID == imageID
	})
	assert.NotEmpty(t, app.Status.Namespace)
	assert.Equal(t, app.Status.AppImage.ID, imageID)

	imageID2 := client2.NewImage2(t, project.Name)
	err = c.ImageTag(ctx, imageID2, "foo:latest")
	if err != nil {
		t.Fatal(err)
	}

	err = c.AppPullImage(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppImage.ID == imageID2
	})
	assert.Equal(t, app.Status.AppImage.ID, imageID2)
}
