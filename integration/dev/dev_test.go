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
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/dev"
	hclient "github.com/acorn-io/acorn/pkg/k8sclient"
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
	c := helper.MustReturn(hclient.Default)
	ns := helper.TempNamespace(t, c)
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

	appWatcher := watcher.New[*v1.AppInstance](c)

	eg := errgroup.Group{}
	eg.Go(func() error {
		return dev.Dev(subCtx, acornCueFile, &dev.Options{
			Client: helper.BuilderClient(t, ns.Name),
			Build: build.Options{
				Cwd:    tmp,
			},
			Run: client.AppRunOptions{
				Name:      "test-app",
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

	_, err = appWatcher.ByName(ctx, ns.Name, "test-app", func(app *v1.AppInstance) (bool, error) {
		return app.Spec.Image != oldImage && app.Spec.Image != "", nil
	})
	if err != nil {
		t.Fatal(err)
	}

	cancel()
	_, err = appWatcher.ByName(ctx, ns.Name, "test-app", func(app *v1.AppInstance) (bool, error) {
		return app.Spec.Stop != nil && *app.Spec.Stop, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := eg.Wait(); err != nil {
		t.Fatal(err)
	}
}
