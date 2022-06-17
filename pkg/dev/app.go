package dev

import (
	"context"
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	objwatcher "github.com/acorn-io/acorn/pkg/watcher"
)

func appPrintLoop(ctx context.Context, app *apiv1.App, opts *Options) error {
	w := objwatcher.New[*apiv1.App](opts.Client.GetClient())
	_, err := w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		fmt.Printf("STATUS: ENDPOINTS[%s] HEALTHY[%s] UPTODATE[%s] %s\n",
			app.Status.Columns.Endpoints,
			app.Status.Columns.Healthy,
			app.Status.Columns.UpToDate,
			app.Status.Columns.Message)
		return false, nil
	})
	return err
}

func appStatusLoop(ctx context.Context, apps <-chan *apiv1.App, opts *Options) error {
	var (
		displayCancel = func() {}
		subCtx        context.Context
	)
	for app := range apps {
		displayCancel()
		subCtx, displayCancel = context.WithCancel(ctx)
		app := app
		go func() { _ = appPrintLoop(subCtx, app, opts) }()
	}

	displayCancel()
	return nil
}
