package dev

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	api "github.com/acorn-io/runtime/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/log"
	"github.com/acorn-io/runtime/pkg/rulerequest"
	"github.com/acorn-io/z"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
)

type Logger interface {
	client.ContainerLogsWriter
	AppStatusLogger

	Errorf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

type AppStatusLogger interface {
	AppStatus(ready bool, msg string)
}

type Options struct {
	ImageSource       imagesource.ImageSource
	Run               client.AppRunOptions
	Replace           bool
	Dangerous         bool
	BidirectionalSync bool
	Logger            Logger
	BuildStatus       chan<- BuildStatus
}

type BuildState string

var (
	BuildRunning   = BuildState("running")
	BuildFailed    = BuildState("running")
	BuildSucceeded = BuildState("succeeded")
)

type BuildStatus struct {
	AppName string
	State   BuildState
	Image   string
	Message string
}

func (o *Options) complete() *Options {
	var cp Options
	if o != nil {
		cp = *o
	}
	if cp.Logger == nil {
		cp.Logger = log.DefaultLogger
	}

	return &cp
}

type watcher struct {
	c            client.Client
	imageAndArgs imagesource.ImageSource
	trigger      chan struct{}
	watching     []string
	watchingTS   []time.Time
	initOnce     sync.Once
	logger       Logger
}

func (w *watcher) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *watcher) readFiles(ctx context.Context) ([]string, error) {
	return w.imageAndArgs.WatchFiles(ctx, w.c)
}

func (w *watcher) foundChanges() bool {
	logrus.Tracef("Checking timestamp of %v", w.watching)
	for i, f := range w.watching {
		s, err := os.Stat(f)
		if err == nil {
			if w.watchingTS[i] != s.ModTime() {
				if !w.watchingTS[i].IsZero() {
					w.logger.Infof("%s has changed", f)
				}
				return true
			}
		} else if !os.IsNotExist(err) {
			w.logger.Errorf("failed to read %s: %v", f, err)
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
	files, err := w.readFiles(ctx)
	if err == nil {
		w.watching = files
	} else {
		w.logger.Errorf("failed to resolve files to watch: %v", err)
	}
	w.watchingTS = timestamps(w.watching)
}

func (w *watcher) Wait(ctx context.Context) error {
	init := false
	w.initOnce.Do(func() {
		w.watching, _ = w.readFiles(ctx)
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

func buildFailed(c chan<- BuildStatus, msg string) {
	if c == nil {
		return
	}
	c <- BuildStatus{
		State:   BuildFailed,
		Message: msg,
	}
}

func buildAppName(c chan<- BuildStatus, appName string) {
	if c == nil {
		return
	}
	c <- BuildStatus{
		AppName: appName,
	}
}

func buildSuccess(c chan<- BuildStatus, image string) {
	if c == nil {
		return
	}
	c <- BuildStatus{
		State: BuildSucceeded,
		Image: image,
	}
}

func buildStart(c chan<- BuildStatus) {
	if c == nil {
		return
	}
	c <- BuildStatus{
		State: BuildRunning,
	}
}

func buildLoop(ctx context.Context, c client.Client, hash clientHash, opts *Options) error {
	opts = opts.complete()

	var (
		watcher = watcher{
			trigger:      make(chan struct{}, 1),
			watchingTS:   make([]time.Time, 1),
			imageAndArgs: opts.ImageSource,
			logger:       opts.Logger,
		}
		startLock sync.Mutex
		started   = false
		appName   string
		lockOnce  sync.Once
		logger    = opts.Logger
	)

	defer func() {
		if err := releaseDevSession(c, appName); err != nil {
			logger.Errorf("Failed to release dev session app: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	failed := atomic.Bool{}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Minute):
			}
			if failed.Swap(false) {
				watcher.Trigger()
			}
		}
	}()

	for {
		if err := watcher.Wait(ctx); err != nil {
			return err
		}

		buildStart(opts.BuildStatus)
		image, deployArgs, profiles, err := opts.ImageSource.GetImageAndDeployArgs(ctx, c)
		if err == pflag.ErrHelp {
			continue
		} else if err != nil {
			buildFailed(opts.BuildStatus, err.Error())
			_, buildFile, _ := opts.ImageSource.ResolveImageAndFile()
			if buildFile == "" {
				return err
			}
			logger.Errorf("Failed to build %s: %v", buildFile, err)
			logger.Infof("Build failed, touch [%s] to rebuild", buildFile)
			failed.Store(true)
			continue
		}

		buildSuccess(opts.BuildStatus, image)
		failed.Store(false)

		for {
			appName, err = runOrUpdate(ctx, c, hash, image, deployArgs, profiles, opts)
			if apierror.IsConflict(err) {
				logger.Errorf("Failed to run/update app: %v", err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Second):
					continue
				}
			} else if err != nil {
				logger.Errorf("Failed to run/update app: %v", err)
				failed.Store(true)
			}
			// appName will be empty if the runOrUpdate call fails so wait until first success to start devsession
			if appName != "" {
				buildAppName(opts.BuildStatus, appName)
				lockOnce.Do(func() {
					go func() {
						renewDevSession(ctx, c, logger, appName, hash.Client)
						cancel()
					}()
				})
			}
			break
		}

		if appName == "" {
			// Something failed, so continue and wait
			continue
		}

		startLock.Lock()
		if started {
			startLock.Unlock()
			continue
		}

		opts.Run.Name = appName
		eg, ctx := errgroup.WithContext(ctx)
		eg.Go(func() error {
			return DevPorts(ctx, c, appName)
		})
		eg.Go(func() error {
			return LogLoop(ctx, c, appName, &client.LogOptions{
				Logger: logger,
			})
		})
		eg.Go(func() error {
			return AppStatusLoop(ctx, c, logger, appName)
		})
		eg.Go(func() error {
			return containerSyncLoop(ctx, c, appName, opts)
		})
		eg.Go(func() error {
			return appDeleteStop(ctx, c, logger, appName, cancel)
		})
		go func() {
			err := eg.Wait()
			if err != nil {
				logger.Errorf("dev loop terminated, restarting: %v", err)
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

func updateApp(ctx context.Context, c client.Client, appName string, client v1.DevSessionInstanceClient, image string, deployArgs map[string]any, profiles []string, opts *Options) (_ string, err error) {
	update := opts.Run.ToUpdate()
	update.DevSessionClient = &client
	update.Image = image
	update.DeployArgs = deployArgs
	update.Profiles = profiles
	update.Replace = opts.Replace
	update.Stop = new(bool)
	update.AutoUpgrade = new(bool)
	opts.Logger.Infof("Updating acorn [%s] to image [%s]", appName, image)
	app, err := rulerequest.PromptUpdate(ctx, c, opts.Dangerous, appName, update)
	if err != nil {
		return "", err
	}
	opts.Run.Permissions = app.Spec.GrantedPermissions
	return app.Name, nil
}

func createApp(ctx context.Context, client client.Client, hash clientHash, image string, deployArgs map[string]any, profiles []string, opts *Options) (string, error) {
	opts.Run.Labels = append(opts.Run.Labels,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppDevHash,
			Value:        hash.Hash,
		})

	opts.Run.Annotations = append(opts.Run.Annotations,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppDevHash,
			Value:        hash.Hash,
		})

	runArgs := opts.Run
	runArgs.DeployArgs = deployArgs
	runArgs.Profiles = profiles
	runArgs.Stop = z.Pointer(true)

	app, err := rulerequest.PromptRun(ctx, client, opts.Dangerous, image, runArgs)
	if err != nil {
		return "", err
	}
	return app.Name, nil
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

func getExistingApp(ctx context.Context, client client.Client, opts *Options) (string, error) {
	name := opts.Run.Name
	if name == "" {
		return "", apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "apps",
		}, name)
	}

	app, err := client.AppGet(ctx, name)
	if err != nil {
		return "", err
	}

	return app.Name, nil
}

func releaseDevSession(c client.Client, appName string) error {
	if appName == "" {
		return nil
	}

	// Don't use a passed context, because it will be canceled already
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return c.DevSessionRelease(ctx, appName)
}

func runOrUpdate(ctx context.Context, client client.Client, hash clientHash, image string, deployArgs map[string]any, profiles []string, opts *Options) (string, error) {
	appName, err := getExistingApp(ctx, client, opts)
	if apierror.IsNotFound(err) {
		appName, err = createApp(ctx, client, hash, image, deployArgs, profiles, opts)
	}
	if err != nil {
		return "", err
	}
	return updateApp(ctx, client, appName, hash.Client, image, deployArgs, profiles, opts)
}

func appDeleteStop(ctx context.Context, c client.Client, logger AppStatusLogger, appName string, cancel func()) error {
	wc, err := c.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.App](wc)
	_, err = w.ByName(ctx, c.GetNamespace(), appName, func(app *apiv1.App) (bool, error) {
		if !app.DeletionTimestamp.IsZero() {
			logger.AppStatus(false, fmt.Sprintf("app %s deleted, exiting", app.Name))
			cancel()
			return true, nil
		}
		return false, nil
	})
	return err
}

func renewDevSession(ctx context.Context, c client.Client, logger Logger, appName string, client v1.DevSessionInstanceClient) {
	timeout := 20 * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout):
		}

		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return c.DevSessionRenew(ctx, appName, client)
		})
		if apierror.IsNotFound(err) {
			logger.Errorf("Dev session lost [%s]: %v", appName, err)
			return
		} else if err == nil {
			timeout = 20 * time.Second
		} else {
			timeout = 5 * time.Second
			logger.Errorf("Failed to lock acorn [%s]: %v", appName, err)
		}
	}
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

func PrintAppStatus(app *apiv1.App, logger AppStatusLogger) {
	msg, ready := appStatusMessage(app)
	logger.AppStatus(ready, msg)
}

func AppStatusLoop(ctx context.Context, c client.Client, logger AppStatusLogger, appName string) error {
	wc, err := c.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.App](wc)
	msg, ready := "", false
	_, err = w.ByName(ctx, c.GetNamespace(), appName, func(app *apiv1.App) (bool, error) {
		newMsg, newReady := appStatusMessage(app)
		logrus.Debugf("app status loop %s/%s rev=%s, generation=%d, observed=%d: newMsg=%s, newReady=%v", app.Namespace, app.Name,
			app.ResourceVersion, app.Generation, app.Status.ObservedGeneration, newMsg, newReady)
		if newMsg != msg || newReady != ready {
			PrintAppStatus(app, logger)
		}
		msg, ready = newMsg, newReady

		// Return false because the context will be canceled when this check should stop.
		return false, nil
	})
	return err
}

func LogLoop(ctx context.Context, c client.Client, appName string, opts *client.LogOptions) error {
	for {
		if opts == nil {
			opts = &client.LogOptions{}
		}
		opts.Follow = true
		_ = log.Output(ctx, c, appName, opts)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

type clientHash struct {
	Client v1.DevSessionInstanceClient
	Hash   string
}

func setAppNameAndGetHash(ctx context.Context, c client.Client, opts *Options) (clientHash, *Options, error) {
	image, file, err := opts.ImageSource.ResolveImageAndFile()
	if err != nil {
		return clientHash{}, nil, err
	}
	hostname, _ := os.Hostname()
	hash := client.BuildClientID(image, file)

	if opts.Run.Name == "" {
		existingName, err := getAppName(ctx, c, hash)
		if err != nil {
			return clientHash{}, nil, err
		}
		opts.Run.Name = existingName
	}

	return clientHash{
		Client: v1.DevSessionInstanceClient{
			Hostname: hostname,
			ImageSource: v1.DevSessionImageSource{
				Image: image,
				File:  file,
			},
		},
		Hash: hash,
	}, opts, nil
}

func Dev(ctx context.Context, client client.Client, opts *Options) error {
	hash, opts, err := setAppNameAndGetHash(ctx, client, opts)
	if err != nil {
		return err
	}

	optsCopy := *opts
	optsCopy.ImageSource.Args = append([]string{"--profile=devMode?"}, opts.ImageSource.Args...)

	err = buildLoop(ctx, client, hash, &optsCopy)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
