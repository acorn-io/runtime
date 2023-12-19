package resolvedofferings

import (
	"context"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddDefaultRegion(ctx context.Context, c client.Client, app *v1.AppInstance) error {
	if app.GetRegion() == "" {
		project := new(v1.ProjectInstance)
		if err := c.Get(ctx, client.ObjectKey{Name: app.GetNamespace()}, project); err != nil {
			return err
		}

		app.Status.ResolvedOfferings.Region = project.Status.DefaultRegion
	} else {
		app.Status.ResolvedOfferings.Region = app.GetRegion()
	}

	return nil
}
