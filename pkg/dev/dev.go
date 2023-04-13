package dev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/digest"
	"github.com/acorn-io/acorn/pkg/imagesource"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/log"
	"github.com/acorn-io/acorn/pkg/rulerequest"
	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	"github.com/pterm/pterm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Options struct {
	ImageSource       imagesource.ImageSource
	Run               client.AppRunOptions
	Replace           bool
	Dangerous         bool
	BidirectionalSync bool
}

type watcher struct {
	c            client.Client
	imageAndArgs imagesource.ImageSource
	trigger      chan struct{}
	watching     []string
	watchingTS   []time.Time
	initOnce     sync.Once
}

func (w *watcher) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *watcher) readFiles(ctx context.Context) []string {
	files, err := w.imageAndArgs.WatchFiles(ctx, w.c)
	if err != nil {
		logrus.Errorf("failed to resolve files to watch: %v", err)
	}
	return files
}

func (w *watcher) foundChanges() bool {
	logrus.Debugf("Checking timestamp of %v", w.watching)
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

func (w *watcher) updateTimestamps(ctx context.Context) {
	files := w.readFiles(ctx)
	w.watching = files
	w.watchingTS = timestamps(files)
}

func (w *watcher) Wait(ctx context.Context) error {
	init := false
	w.initOnce.Do(func() {
		w.watching = w.readFiles(ctx)
		init = true
	})

	for {
		if !init && !w.foundChanges() {
			select {
			case <-w.trigger:
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				continue
			}
		}

		w.updateTimestamps(ctx)
		return nil
	}
}

func buildLoop(ctx context.Context, client client.Client, hash string, opts *Options) error {
	defer func() {
		if err := stop(client, opts); err != nil {
			logrus.Errorf("Failed to stop app: %v", err)
		}
	}()

	var (
		watcher = watcher{
			trigger:      make(chan struct{}, 1),
			watchingTS:   make([]time.Time, 1),
			imageAndArgs: opts.ImageSource,
		}
		startLock sync.Mutex
		started   = false
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

outer:
	for {
		if err := watcher.Wait(ctx); err != nil {
			return err
		}

		image, deployArgs, err := opts.ImageSource.GetImageAndDeployArgs(ctx, client)
		if err == pflag.ErrHelp {
			continue
		} else if err != nil {
			_, buildFile, _ := opts.ImageSource.ResolveImageAndFile()
			if buildFile == "" {
				return err
			}
			logrus.Errorf("Failed to build %s: %v", buildFile, err)
			logrus.Infof("Build failed, touch [%s] to rebuild", buildFile)
			go func() {
				time.Sleep(120 * time.Second)
				watcher.Trigger()
			}()
			continue
		}

		var (
			app *apiv1.App
		)
		for {
			app, err = runOrUpdate(ctx, client, hash, image, deployArgs, opts)
			if apierror.IsConflict(err) {
				logrus.Errorf("Failed to run/update app: %v", err)
				time.Sleep(time.Second)
				continue
			} else if err != nil {
				logrus.Errorf("Failed to run/update app: %v", err)
				continue outer
			}
			break
		}

		startLock.Lock()
		if started {
			startLock.Unlock()
			continue
		}

		opts.Run.Name = app.Name
		eg, ctx := errgroup.WithContext(ctx)
		eg.Go(func() error {
			return LogLoop(ctx, client, app, nil)
		})
		eg.Go(func() error {
			return AppStatusLoop(ctx, client, app)
		})
		eg.Go(func() error {
			return containerSyncLoop(ctx, client, app, opts)
		})
		eg.Go(func() error {
			return appDeleteStop(ctx, client, app, cancel)
		})
		go func() {
			err := eg.Wait()
			if err != nil {
				logrus.Error("dev loop terminated, restarting: ", err)
			}
			startLock.Lock()
			started = false
			startLock.Unlock()
			watcher.Trigger()
		}()

		started = true
		startLock.Unlock()
	}
}

func updateApp(ctx context.Context, c client.Client, app *apiv1.App, image string, opts *Options) (err error) {
	defer func() {
		if err == nil && app.Spec.Stop != nil && *app.Spec.Stop {
			err = c.AppStart(ctx, app.Name)
		}
	}()
	// It is possible the the current app.spec.image points to something that is missing, so we need to ensure
	// we update the app.spec.image before we touch anything else, like app.Spec.Stop.  That is why start is in
	// the defer above
	update := opts.Run.ToUpdate()
	update.Image = image
	update.Replace = opts.Replace
	logrus.Infof("Updating app [%s] to image [%s]", app.Name, image)
	app, err = rulerequest.PromptUpdate(ctx, c, opts.Dangerous, app.Name, update)
	return err
}

func createApp(ctx context.Context, client client.Client, hash, image string, opts *Options) (*apiv1.App, error) {
	opts.Run.Labels = append(opts.Run.Labels,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppDevHash,
			Value:        hash,
		})

	opts.Run.Annotations = append(opts.Run.Annotations,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppDevHash,
			Value:        hash,
		})

	app, err := rulerequest.PromptRun(ctx, client, opts.Dangerous, image, opts.Run)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func getAppName(ctx context.Context, client client.Client, hash string) (string, error) {
	apps, err := client.AppList(ctx)
	if err != nil {
		return "", err
	}

	for _, app := range apps {
		if app.Labels[labels.AcornAppDevHash] == hash {
			return app.Name, nil
		}
	}

	return "", nil
}

func getExistingApp(ctx context.Context, client client.Client, opts *Options) (*apiv1.App, error) {
	name := opts.Run.Name
	if name == "" {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "apps",
		}, name)
	}

	return client.AppGet(ctx, name)
}

func stop(c client.Client, opts *Options) error {
	// Don't use a passed context, because it will be canceled already
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	existingApp, err := getExistingApp(ctx, c, opts)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	_, _ = c.AppUpdate(ctx, existingApp.Name, &client.AppUpdateOptions{
		DevMode: new(bool),
	})
	return c.AppStop(ctx, existingApp.Name)
}

func runOrUpdate(ctx context.Context, client client.Client, hash, image string, deployArgs map[string]any, opts *Options) (*apiv1.App, error) {
	opts.Run.DeployArgs = deployArgs
	opts.Run.DevMode = &[]bool{true}[0]
	existingApp, err := getExistingApp(ctx, client, opts)
	if apierror.IsNotFound(err) {
		return createApp(ctx, client, hash, image, opts)
	} else if err != nil {
		return nil, err
	}
	return existingApp, updateApp(ctx, client, existingApp, image, opts)
}

func appDeleteStop(ctx context.Context, c client.Client, app *apiv1.App, cancel func()) error {
	wc, err := c.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.App](wc)
	_, err = w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		if !app.DeletionTimestamp.IsZero() {
			pterm.Println(pterm.FgCyan.Sprintf("app %s deleted, exiting", app.Name))
			cancel()
			return true, nil
		}
		if app.Spec.Stop != nil && *app.Spec.Stop {
			pterm.Println(pterm.FgCyan.Sprintf("starting app %s", app.Name))
			_ = c.AppStart(ctx, app.Name)
		}
		return false, nil
	})
	return err
}

func appStatusMessage(app *apiv1.App) (string, bool) {
	ready := app.Status.Ready && app.Generation == app.Status.ObservedGeneration
	msg := app.Status.Columns.Message
	if !ready && msg == "OK" {
		// This is really only needed on the first run, before the controller runs
		msg = "pending"
	}
	return fmt.Sprintf("STATUS: ENDPOINTS[%s] HEALTHY[%s] UPTODATE[%s] %s",
		app.Status.Columns.Endpoints,
		app.Status.Columns.Healthy,
		app.Status.Columns.UpToDate,
		msg), ready
}

func PrintAppStatus(app *apiv1.App) {
	msg, ready := appStatusMessage(app)
	if ready {
		pterm.DefaultBox.Println(pterm.LightGreen(msg))
	} else {
		pterm.Println(pterm.LightYellow(msg))
	}
}

func AppStatusLoop(ctx context.Context, c client.Client, app *apiv1.App) error {
	wc, err := c.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.App](wc)
	msg, ready := "", false
	_, err = w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		newMsg, newReady := appStatusMessage(app)
		logrus.Debugf("app status loop %s/%s rev=%s, generation=%d, observed=%d: newMsg=%s, newReady=%v", app.Namespace, app.Name,
			app.ResourceVersion, app.Generation, app.Status.ObservedGeneration, newMsg, newReady)
		if newMsg != msg || newReady != ready {
			PrintAppStatus(app)
		}
		msg, ready = newMsg, newReady

		// Return false because the context will be canceled when this check should stop.
		return false, nil
	})
	return err
}

func LogLoop(ctx context.Context, c client.Client, app *apiv1.App, opts *client.LogOptions) error {
	for {
		if opts == nil {
			opts = &client.LogOptions{}
		}
		opts.Follow = true
		_ = log.Output(ctx, c, app.Name, opts)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func setAppNameAndGetHash(ctx context.Context, client client.Client, opts *Options) (string, *Options, error) {
	image, hashSource, err := opts.ImageSource.ResolveImageAndFile()
	if err != nil {
		return "", nil, err
	}
	if hashSource == "" {
		hashSource = image
	}
	cwd, _ := os.Getwd()
	hostname, _ := os.Hostname()
	hashSource = filepath.Join(cwd, hashSource)
	hash := digest.SHA256(hostname, hashSource)[:12]

	if opts.Run.Name == "" {
		existingName, err := getAppName(ctx, client, hash)
		if err != nil {
			return "", nil, err
		}
		opts.Run.Name = existingName
	}

	return hash, opts, nil
}

func Dev(ctx context.Context, client client.Client, opts *Options) error {
	hash, opts, err := setAppNameAndGetHash(ctx, client, opts)
	if err != nil {
		return err
	}

	opts.Run.Profiles = append([]string{"dev?"}, opts.Run.Profiles...)
	opts.ImageSource.Profiles = append([]string{"dev?"}, opts.ImageSource.Profiles...)

	err = buildLoop(ctx, client, hash, opts)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
