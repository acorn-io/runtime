package dev

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/pkg/deployargs"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Options struct {
	Args   []string
	Client client.Client
	Build  build.Options
	Run    client.AppRunOptions
	Log    client.LogOptions
}

func (o *Options) Complete() (*Options, error) {
	var (
		result Options
		err    error
	)

	if o != nil {
		result = *o
	}

	if result.Client == nil {
		result.Client, err = client.Default()
		if err != nil {
			return nil, err
		}
	}

	result.Build.Client = result.Client
	return &result, nil
}

type watcher struct {
	file       string
	cwd        string
	trigger    <-chan struct{}
	watching   []string
	watchingTS []time.Time
}

func (w *watcher) readFiles() []string {
	data, err := cue.ReadCUE(w.file)
	if err != nil {
		logrus.Errorf("failed to read %s: %v", w.file, err)
		return []string{w.file}
	}
	app, err := appdefinition.NewAppDefinition(data)
	if err != nil {
		logrus.Errorf("failed to parse %s: %v", w.file, err)
		return []string{w.file}
	}
	files, err := app.WatchFiles(w.cwd)
	if err != nil {
		logrus.Errorf("failed to parse additional files %s: %v", w.file, err)
		return []string{w.file}
	}
	return append([]string{w.file}, files...)
}

func (w *watcher) foundChanges() bool {
	for i, f := range w.watching {
		s, err := os.Stat(f)
		if err == nil {
			if w.watchingTS[i] != s.ModTime() {
				if !w.watchingTS[i].IsZero() {
					logrus.Infof("%s has changed", f)
				}
				return true
			}
		} else {
			logrus.Errorf("failed to read %s: %v", f, err)
		}
	}
	return false
}

func timestamps(files []string) []time.Time {
	result := make([]time.Time, len(files))
	for i, f := range files {
		s, err := os.Stat(f)
		if err == nil {
			result[i] = s.ModTime()
		}
	}
	return result
}

func (w *watcher) Wait(ctx context.Context) error {
	for {
		if !w.foundChanges() {
			select {
			case <-w.trigger:
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
				continue
			}
		}

		files := w.readFiles()
		w.watching = files
		w.watchingTS = timestamps(files)
		return nil
	}
}

func buildLoop(ctx context.Context, file string, opts *build.Options, trigger <-chan struct{}, result chan<- string) error {
	var (
		watcher = watcher{
			file:       file,
			cwd:        opts.Cwd,
			trigger:    typed.Debounce(trigger),
			watching:   []string{file},
			watchingTS: make([]time.Time, 1),
		}
	)

	for {
		if err := watcher.Wait(ctx); err != nil {
			return err
		}

		image, err := build.Build(ctx, file, opts)
		if err != nil {
			logrus.Errorf("Failed to build %s: %v", file, err)
			continue
		}

		result <- image.ID
	}
}

func getPathHash(acornCue string) string {
	sum := sha256.Sum256([]byte(acornCue))
	return hex.EncodeToString(sum[:])[:12]
}

func updateApp(ctx context.Context, c client.Client, app *apiv1.App, image string, opts *Options) error {
	if app.Spec.Stop != nil && *app.Spec.Stop {
		err := c.AppStart(ctx, app.Name)
		if err != nil {
			return err
		}
	}
	update := opts.Run.ToUpdate()
	update.Image = image
	_, err := c.AppUpdate(ctx, app.Name, &update)
	return err
}

func createApp(ctx context.Context, acornCue, image string, opts *Options, apps chan<- *apiv1.App) (string, error) {
	if opts.Run.Labels == nil {
		opts.Run.Labels = map[string]string{}
	}
	opts.Run.Labels[labels.AcornAppCuePath] = getPathHash(acornCue)

	if opts.Run.Annotations == nil {
		opts.Run.Annotations = map[string]string{}
	}
	opts.Run.Annotations[labels.AcornAppCuePath] = acornCue

	app, err := opts.Client.AppRun(ctx, image, &opts.Run)
	if err != nil {
		return "", err
	}
	apps <- app
	return app.Name, nil
}

func getAppName(ctx context.Context, acornCue string, opts *Options) (string, error) {
	apps, err := opts.Client.AppList(ctx)
	if err != nil {
		return "", err
	}

	hash := getPathHash(acornCue)
	for _, app := range apps {
		if app.Labels[labels.AcornAppCuePath] == hash {
			return app.Name, nil
		}
	}

	return "", nil
}

func getExistingApp(ctx context.Context, opts *Options) (*apiv1.App, error) {
	name := opts.Run.Name
	if name == "" {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "apps",
		}, name)
	}

	return opts.Client.AppGet(ctx, name)
}

func stop(opts *Options) error {
	// Don't use a passed context, because it will be canceled already
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	existingApp, err := getExistingApp(ctx, opts)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	return opts.Client.AppStop(ctx, existingApp.Name)
}

func runOrUpdate(ctx context.Context, acornCue, image string, opts *Options, apps chan<- *apiv1.App) (string, error) {
	_, flags, err := deployargs.ToFlagsFromImage(ctx, opts.Client, image)
	if err != nil {
		return "", err
	}

	if len(opts.Args) > 0 {
		deployArgs, err := flags.Parse(opts.Args)
		if err != nil {
			return "", err
		}
		opts.Run.DeployArgs = deployArgs
	}

	existingApp, err := getExistingApp(ctx, opts)
	if apierror.IsNotFound(err) {
		return createApp(ctx, acornCue, image, opts, apps)
	} else if err != nil {
		return "", err
	}
	apps <- existingApp
	return existingApp.Name, updateApp(ctx, opts.Client, existingApp, image, opts)
}

func runLoop(ctx context.Context, acornCue string, opts *Options, images <-chan string, apps chan<- *apiv1.App) error {
	defer func() {
		if err := stop(opts); err != nil {
			logrus.Errorf("Failed to stop app: %v", err)
		}
	}()
	for {
		select {
		case image, open := <-images:
			if !open {
				return nil
			}
			if newName, err := runOrUpdate(ctx, acornCue, image, opts, apps); err != nil {
				logrus.Errorf("Failed to run app: %v", err)
			} else {
				opts.Run.Name = newName
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func doLog(ctx context.Context, app *apiv1.App, opts *Options) <-chan error {
	result := make(chan error, 1)
	go func() {
		fmt.Println("Watching logs for", app.Name)
		opts.Log.Follow = true
		result <- log.Output(ctx, opts.Client, app.Name, &opts.Log)
		fmt.Println("Terminating logging for", app.Name)
	}()
	return result
}

func logLoop(ctx context.Context, apps <-chan *apiv1.App, opts *Options) error {
	var (
		logging = false
		logChan <-chan error
		lastApp *apiv1.App
		cancel  = func() {}
		logCtx  context.Context
	)

	defer cancel()

	for {
		select {
		case <-ctx.Done():
			cancel()
			return ctx.Err()
		case <-logChan:
			if lastApp == nil {
				logging = false
			} else {
				cancel()
				logCtx, cancel = context.WithCancel(ctx)
				logChan = doLog(logCtx, lastApp, opts)
			}
		case app, open := <-apps:
			if !open {
				cancel()
				return nil
			}
			if logging && lastApp.Name == app.Name {
				continue
			}
			lastApp = app
			logging = true
			cancel()
			logCtx, cancel = context.WithCancel(ctx)
			logChan = doLog(logCtx, lastApp, opts)
		}
	}
}

func readInput(ctx context.Context, trigger chan<- struct{}) error {
	readSomething := make(chan struct{})

	go func() {
		line := bufio.NewScanner(os.Stdin)
		for line.Scan() {
			if strings.Contains(line.Text(), "b") {
				readSomething <- struct{}{}
			}
		}
		<-ctx.Done()
		close(readSomething)
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-readSomething:
			if !ok {
				<-ctx.Done()
				return nil
			}
			trigger <- struct{}{}
		}
	}
}

func resolveAcornCueAndName(ctx context.Context, acornCue string, opts *Options) (string, *Options, error) {
	nameWasSet := opts.Run.Name != ""
	opts, err := opts.Complete()
	if err != nil {
		return "", nil, err
	}

	acornCue = build.ResolveFile(acornCue, opts.Build.Cwd)

	if !filepath.IsAbs(acornCue) {
		acornCue, err = filepath.Abs(acornCue)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve the location of %s: %w", acornCue, err)
		}
	}

	if !nameWasSet {
		existingName, err := getAppName(ctx, acornCue, opts)
		if err != nil {
			return "", nil, err
		}
		opts.Run.Name = existingName
	}

	return acornCue, opts, nil
}

func Dev(ctx context.Context, file string, opts *Options) error {
	acornCue, opts, err := resolveAcornCueAndName(ctx, file, opts)
	if err != nil {
		return err
	}

	trigger := make(chan struct{}, 1)
	images := make(chan string, 1)
	apps := make(chan *apiv1.App, 1)
	appLogs, appStatus := typed.Tee(apps)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return readInput(ctx, trigger)
	})
	eg.Go(func() error {
		defer close(images)
		return buildLoop(ctx, acornCue, &opts.Build, trigger, images)
	})
	eg.Go(func() error {
		defer close(apps)
		return runLoop(ctx, acornCue, opts, images, apps)
	})
	eg.Go(func() error {
		return logLoop(ctx, appLogs, opts)
	})
	eg.Go(func() error {
		return appStatusLoop(ctx, appStatus, opts)
	})
	err = eg.Wait()
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
