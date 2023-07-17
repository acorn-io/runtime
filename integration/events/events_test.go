package events

import (
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppEvents(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	c, project := helper.ClientAndProject(t)

	image, err := c.AcornImageBuild(ctx, "./testdata/simple/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create the app to generate an AppCreateEvent
	app, err := c.AppRun(ctx, image.ID, nil)
	if err != nil {
		t.Fatal(err)
	}

	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		return obj.Status.Condition(v1.AppInstanceConditionParsed).Success
	})
	assert.NotEmpty(t, app.Status.Namespace)

	// Update the app spec to generate an AppSpecUpdatedEvent
	kc, err := c.GetClient()
	require.NoError(t, err)

	stop := true
	app = helper.WaitForObject(t, helper.Watcher(t, c), &apiv1.AppList{}, app, func(obj *apiv1.App) bool {
		obj.Spec.Stop = &stop
		return kc.Update(ctx, obj) == nil
	})

	// Delete the app to generate an AppDeletedEvent
	assert.NoError(t, kc.Delete(ctx, app))

	// Ensure an event of each type has been recorded
	var created, updated, deleted bool
	helper.Wait(t, helper.Watcher(t, c), &apiv1.EventList{}, func(obj *apiv1.Event) bool {
		if obj.Namespace != project.Name || obj.Resource.Kind != "app" || obj.Resource.Name != app.Name {
			// This event isn't for our app
			return false
		}

		switch obj.Type {
		case apps.AppCreateEventType:
			created = true
		case apps.AppSpecUpdateEventType:
			updated = true
		case apps.AppDeleteEventType:
			deleted = true
		}

		return created && updated && deleted
	})
}
