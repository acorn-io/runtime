package wait

import (
	"context"
	"fmt"
	"sync"

	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/dev"
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
			_ = dev.AppStatusLoop(ctx, c, app.Name)
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
	wc, err := c.GetClient()
	if err != nil {
		return nil, err
	}
	w := objwatcher.New[*apiv1.App](wc)
	return w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		if app.Status.Ready && app.Generation == app.Status.ObservedGeneration {
			return true, nil
		}
		for name, job := range app.Status.AppStatus.Jobs {
			if !job.Ready && job.RunningCount == 0 && job.ErrorCount > 0 && len(job.ErrorMessages) > 0 {
				return false, fmt.Errorf("job %s failed: %s", name, job.ErrorMessages)
			}
		}
		return false, nil
	})
}
