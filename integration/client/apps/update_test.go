package apps

import (
	"testing"

	client2 "github.com/acorn-io/acorn/integration/client"
	"github.com/acorn-io/acorn/integration/helper"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePull(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppRun(ctx, "foo", &client.AppRunOptions{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppImage.ID == imageID
	})
	assert.NotEmpty(t, app.Status.Namespace)
	assert.Equal(t, app.Status.AppImage.ID, imageID)

	imageID2 := client2.NewImage2(t, ns.Name)
	err = c.ImageTag(ctx, imageID2, "foo")
	if err != nil {
		t.Fatal(err)
	}

	err = c.AppPullImage(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, c.GetClient().Watch, &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.AppImage.ID == imageID2
	})
	assert.Equal(t, app.Status.AppImage.ID, imageID2)
}
