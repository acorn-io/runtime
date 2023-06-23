package services

import (
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
)

func TestServices(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, _ := helper.ClientAndNamespace(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata",
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
