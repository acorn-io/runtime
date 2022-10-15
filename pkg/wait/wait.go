package wait

import (
	"context"
	"fmt"
	"sync"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/dev"
	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
)

func App(ctx context.Context, c client.Client, appName string, quiet bool) error {
	app, err := c.AppGet(ctx, appName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg := sync.WaitGroup{}

	if !quiet {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = dev.AppStatusLoop(ctx, c, app)
		}()
	}

	app, err = waitForApp(ctx, c, app)
	if err != nil {
		return err
	}

	cancel()
	wg.Wait()
	fmt.Println()
	dev.PrintAppStatus(app)
	return nil
}

func waitForApp(ctx context.Context, c client.Client, app *apiv1.App) (*apiv1.App, error) {
	w := objwatcher.New[*apiv1.App](c.GetClient())
	return w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
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
}
