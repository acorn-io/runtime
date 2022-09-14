package wait

import (
	"context"
	"fmt"
	"sync"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/dev"
	objwatcher "github.com/acorn-io/acorn/pkg/watcher"
)

func App(ctx context.Context, c client.Client, appName string, quiet bool) error {
	app, err := c.AppGet(ctx, appName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := sync.WaitGroup{}

	defer func() {
		cancel()
		wg.Wait()
		app, err := c.AppGet(ctx, appName)
		if err == nil {
			fmt.Println()
			dev.PrintAppStatus(app)
		}
	}()

	if !quiet {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = dev.AppStatusLoop(ctx, c, app)
		}()
	}

	return waitForApp(ctx, c, app)
}

func waitForApp(ctx context.Context, c client.Client, app *apiv1.App) error {
	w := objwatcher.New[*apiv1.App](c.GetClient())
	_, err := w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		if app.Status.Ready {
			return true, nil
		}
		for name, job := range app.Status.JobsStatus {
			if job.Failed {
				return false, fmt.Errorf("job %s failed: %s", name, job.Message)
			}
		}
		return false, nil
	})
	return err
}
