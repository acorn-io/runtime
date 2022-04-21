package dev

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/dev"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/acorn/pkg/run"
	"github.com/acorn-io/acorn/pkg/watcher"
	"golang.org/x/sync/errgroup"
)

const (
	acornCue = `
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
	tmp, err := ioutil.TempDir("", "acorn-test-dev")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmp)
	})

	acornCueFile := filepath.Join(tmp, "acorn.cue")
	err = ioutil.WriteFile(acornCueFile, []byte(acornCue), 0600)
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
		return dev.Dev(subCtx, acornCueFile, &dev.Options{
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
