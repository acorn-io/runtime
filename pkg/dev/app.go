package dev

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	objwatcher "github.com/acorn-io/acorn/pkg/watcher"
)

func appPrintLoop(ctx context.Context, app *v1.AppInstance, opts *Options) error {
	w := objwatcher.New[*v1.AppInstance](opts.Run.Client)
	_, err := w.ByObject(ctx, app, func(app *v1.AppInstance) (bool, error) {
		fmt.Printf("STATUS: ENDPOINTS[%s] HEALTHY[%s] UPTODATE[%s] %s\n",
			app.Status.Columns.Endpoints,
			app.Status.Columns.Healthy,
			app.Status.Columns.UpToDate,
			app.Status.Columns.Message)
		return false, nil
	})
	return err
}

func appStatusLoop(ctx context.Context, apps <-chan *v1.AppInstance, opts *Options) error {
	var (
		displayCancel = func() {}
		subCtx        context.Context
	)

	for app := range apps {
		displayCancel()
		subCtx, displayCancel = context.WithCancel(ctx)
		go appPrintLoop(subCtx, app, opts)
	}
	return nil
}
