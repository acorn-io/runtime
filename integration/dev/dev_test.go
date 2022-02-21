package dev

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ibuildthecloud/herd/integration/helper"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/build"
	hclient "github.com/ibuildthecloud/herd/pkg/client"
	"github.com/ibuildthecloud/herd/pkg/dev"
	"github.com/ibuildthecloud/herd/pkg/log"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/ibuildthecloud/herd/pkg/watcher"
	"golang.org/x/sync/errgroup"
)

const (
	herdCue = `
containers: default: build: {}
`
	dockerfile1 = `
FROM busybox
CMD ["echo", "hi"]`
	dockerfile2 = `
FROM busybox
CMD ["echo", "bye"]`
)

func TestDev(t *testing.T) {
	helper.StartController(t)
	ctx := helper.GetCTX(t)
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	client := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, client)
	tmp, err := ioutil.TempDir("", "herd-test-dev")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	herdCueFile := filepath.Join(tmp, "herd.cue")
	err = ioutil.WriteFile(herdCueFile, []byte(herdCue), 0600)
	if err != nil {
		t.Fatal(err)
	}

	dockerFile := filepath.Join(tmp, "Dockerfile")
	err = ioutil.WriteFile(dockerFile, []byte(dockerfile1), 0600)
	if err != nil {
		t.Fatal(err)
	}

	appWatcher := watcher.New[*v1.AppInstance](client)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return dev.Dev(subCtx, herdCueFile, &dev.Options{
			Build: build.Options{
				Cwd: tmp,
			},
			Run: run.Options{
				Name:      "test-app",
				Namespace: ns.Name,
				Client:    client,
			},
			Log: log.Options{
				Client: client,
			},
		})
	})

	app, err := appWatcher.ByName(ctx, ns.Name, "test-app", func(app *v1.AppInstance) (bool, error) {
		return app.Spec.Image != "", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	oldImage := app.Spec.Image
	err = ioutil.WriteFile(dockerFile, []byte(dockerfile2), 0600)
	if err != nil {
		t.Fatal(err)
	}

	app, err = appWatcher.ByName(ctx, ns.Name, "test-app", func(app *v1.AppInstance) (bool, error) {
		return app.Spec.Image != oldImage && app.Spec.Image != "", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	app, err = appWatcher.ByName(ctx, ns.Name, "test-app", func(app *v1.AppInstance) (bool, error) {
		return app.Spec.Stop != nil && *app.Spec.Stop, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
