package dev

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ibuildthecloud/baaah/pkg/typed"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/ibuildthecloud/herd/pkg/labels"
	"github.com/ibuildthecloud/herd/pkg/log"
	"github.com/ibuildthecloud/herd/pkg/run"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Build build.Options
	Run   run.Options
	Log   log.Options
}

func (o *Options) Complete() (*Options, error) {
	result := *o

	if result.Run.Client == nil && result.Log.Client != nil {
		result.Run.Client = result.Log.Client
	}

	buildOpts, err := result.Build.Complete()
	if err != nil {
		return nil, err
	}
	result.Build = *buildOpts

	runOpts, err := result.Run.Complete()
	if err != nil {
		return nil, err
	}
	result.Run = *runOpts

	logOpts, err := result.Log.Complete()
	if err != nil {
		return nil, err
	}
	result.Log = *logOpts
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
	data, err := appdefinition.ReadCUE(w.file)
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

func buildLoop(ctx context.Context, file string, opts build.Options, trigger <-chan struct{}, result chan<- string) error {
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

		image, err := build.Build(ctx, file, &opts)
		if err != nil {
			logrus.Errorf("Failed to build %s: %v", file, err)
			continue
		}

		result <- image.ID
	}
}

func getByPathLabels(herdCue string) klabels.Set {
	sum := sha256.Sum256([]byte(herdCue))
	return klabels.Set{
		labels.HerdAppCuePath: hex.EncodeToString(sum[:])[:12],
	}
}

func getByPathSelector(herdCue string) klabels.Selector {
	return klabels.SelectorFromSet(getByPathLabels(herdCue))
}

func updateApp(ctx context.Context, client client.Client, app *v1.AppInstance, image string, opts run.Options) error {
	app.Spec.Image = image
	app.Spec.Stop = new(bool)
	app.Spec.Endpoints = opts.Endpoints
	return client.Update(ctx, app)
}

func createApp(ctx context.Context, herdCue, image string, opts run.Options, apps chan<- *v1.AppInstance) (string, error) {
	if opts.Labels == nil {
		opts.Labels = map[string]string{}
	}
	if opts.Annotations == nil {
		opts.Annotations = map[string]string{}
	}
	for k, v := range getByPathLabels(herdCue) {
		opts.Labels[k] = v
	}
	opts.Annotations[labels.HerdAppCuePath] = herdCue
	app, err := run.Run(ctx, image, &opts)
	if err != nil {
		return "", err
	}
	apps <- app
	return app.Name, nil
}

func getAppName(ctx context.Context, herdCue string, opts run.Options) (string, error) {
	var apps v1.AppInstanceList
	err := opts.Client.List(ctx, &apps, &client.ListOptions{
		LabelSelector: getByPathSelector(herdCue),
		Namespace:     opts.Namespace,
	})
	if err != nil {
		return "", err
	}
	if len(apps.Items) > 0 {
		sort.Slice(apps.Items, func(i, j int) bool {
			return apps.Items[i].Name < apps.Items[j].Name
		})
		return apps.Items[0].Name, nil
	}

	return "", nil
}

func getExistingApp(ctx context.Context, herdCue string, opts run.Options) (*v1.AppInstance, error) {
	name := opts.Name
	if name == "" {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    v1.SchemeGroupVersion.Group,
			Resource: "appinstances",
		}, name)
	}

	var existingApp v1.AppInstance
	err := opts.Client.Get(ctx, client.ObjectKey{Name: name, Namespace: opts.Namespace}, &existingApp)
	return &existingApp, err
}

func stop(herdCue string, opts run.Options) error {
	// Don't use a passed context, because it will be canceled already
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	existingApp, err := getExistingApp(ctx, herdCue, opts)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	if existingApp.Spec.Stop == nil || !*existingApp.Spec.Stop {
		existingApp.Spec.Stop = &[]bool{true}[0]
		return opts.Client.Update(ctx, existingApp)
	}
	return nil
}

func runOrUpdate(ctx context.Context, herdCue, image string, opts run.Options, apps chan<- *v1.AppInstance) (string, error) {
	existingApp, err := getExistingApp(ctx, herdCue, opts)
	if apierror.IsNotFound(err) {
		return createApp(ctx, herdCue, image, opts, apps)
	} else if err != nil {
		return "", err
	}
	apps <- existingApp
	return existingApp.Name, updateApp(ctx, opts.Client, existingApp, image, opts)
}

func runLoop(ctx context.Context, herdCue string, opts run.Options, images <-chan string, apps chan<- *v1.AppInstance) error {
	defer func() {
		if err := stop(herdCue, opts); err != nil {
			logrus.Errorf("Failed to stop app: %v", err)
		}
	}()
	for {
		select {
		case image, open := <-images:
			if !open {
				return nil
			}
			if newName, err := runOrUpdate(ctx, herdCue, image, opts, apps); err != nil {
				logrus.Errorf("Failed to run app: %v", err)
			} else {
				opts.Name = newName
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func doLog(ctx context.Context, app *v1.AppInstance, opts *log.Options) <-chan error {
	result := make(chan error, 1)
	go func() {
		fmt.Println("Watching logs for", app.Name)
		result <- log.Output(ctx, app, opts)
		fmt.Println("Terminating logging for", app.Name)
	}()
	return result
}

func logLoop(ctx context.Context, apps <-chan *v1.AppInstance, opts *log.Options) error {
	var (
		logging = false
		logChan <-chan error
		lastApp *v1.AppInstance
		cancel  = func() {}
		logCtx  context.Context
	)

	for {
		select {
		case <-ctx.Done():
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
				return nil
			}
			if logging && lastApp == app {
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

func resolveHerdCueAndName(ctx context.Context, herdCue string, opts *Options) (string, *Options, error) {
	nameWasSet := opts.Run.Name != ""
	opts, err := opts.Complete()
	if err != nil {
		return "", nil, err
	}

	herdCue = build.ResolveFile(herdCue, opts.Build.Cwd)

	if !filepath.IsAbs(herdCue) {
		herdCue, err = filepath.Abs(herdCue)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve the location of %s: %w", herdCue, err)
		}
	}

	if !nameWasSet {
		existingName, err := getAppName(ctx, herdCue, opts.Run)
		if err != nil {
			return "", nil, err
		}
		opts.Run.Name = existingName
	}

	return herdCue, opts, nil
}

func Dev(ctx context.Context, file string, opts *Options) error {
	herdCue, opts, err := resolveHerdCueAndName(ctx, file, opts)
	if err != nil {
		return err
	}

	trigger := make(chan struct{}, 1)
	images := make(chan string, 1)
	apps := make(chan *v1.AppInstance, 1)
	appLogs, appStatus := typed.Tee(apps)

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return readInput(ctx, trigger)
	})
	eg.Go(func() error {
		defer close(images)
		return buildLoop(ctx, herdCue, opts.Build, trigger, images)
	})
	eg.Go(func() error {
		defer close(apps)
		return runLoop(ctx, herdCue, opts.Run, images, apps)
	})
	eg.Go(func() error {
		return logLoop(ctx, appLogs, &opts.Log)
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
