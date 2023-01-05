package dev

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/cue"
	"github.com/acorn-io/acorn/pkg/deployargs"
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
	Args              []string
	Client            client.Client
	Build             client.AcornImageBuildOptions
	Run               client.AppRunOptions
	Log               client.LogOptions
	Dangerous         bool
	BidirectionalSync bool
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
		dc := client.CmdClient{}
		result.Client, err = dc.CreateDefault()
		if err != nil {
			return nil, err
		}
	}

	return &result, nil
}

type watcher struct {
	file       string
	cwd        string
	args       []string
	trigger    chan struct{}
	watching   []string
	watchingTS []time.Time
}

func (w *watcher) Trigger() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
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
	params, err := build.ParseParams(w.file, w.cwd, w.args)
	if err != nil {
		logrus.Errorf("failed to parse args %v: %v", w.args, err)
		return []string{w.file}
	}
	app, _, err = app.WithArgs(params, []string{"dev?"})
	if err != nil {
		logrus.Errorf("failed to assign args %v: %v", w.args, err)
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
			case <-time.After(time.Second):
				continue
			}
		}

		files := w.readFiles()
		w.watching = files
		w.watchingTS = timestamps(files)
		return nil
	}
}

func buildLoop(ctx context.Context, file string, opts *Options) error {
	defer func() {
		if err := stop(opts); err != nil {
			logrus.Errorf("Failed to stop app: %v", err)
		}
	}()

	var (
		watcher = watcher{
			file:       file,
			cwd:        opts.Build.Cwd,
			trigger:    make(chan struct{}, 1),
			watching:   []string{file},
			watchingTS: make([]time.Time, 1),
			args:       opts.Args,
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

		params, err := build.ParseParams(file, opts.Build.Cwd, opts.Args)
		if err == pflag.ErrHelp {
			continue
		} else if err != nil {
			logrus.Errorf("Failed to parse build args %s: %v", file, err)
			continue
		}

		opts.Build.Args = params
		image, err := opts.Client.AcornImageBuild(ctx, file, &opts.Build)
		if err != nil {
			logrus.Errorf("Failed to build %s: %v", file, err)
			logrus.Infof("Build failed, touch [%s] to rebuild", file)
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
			app, err = runOrUpdate(ctx, file, image.ID, opts)
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
			return LogLoop(ctx, opts.Client, app, &opts.Log)
		})
		eg.Go(func() error {
			return AppStatusLoop(ctx, opts.Client, app)
		})
		eg.Go(func() error {
			return containerSyncLoop(ctx, app, opts)
		})
		eg.Go(func() error {
			return appDeleteStop(ctx, opts.Client, app, cancel)
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
	logrus.Infof("Updating app [%s] to image [%s]", app.Name, image)
	_, err := rulerequest.PromptUpdate(ctx, opts.Client, opts.Dangerous, app.Name, update)
	return err
}

func createApp(ctx context.Context, acornCue, image string, opts *Options) (*apiv1.App, error) {
	opts.Run.Labels = append(opts.Run.Labels,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppCuePath,
			Value:        getPathHash(acornCue),
		})

	opts.Run.Annotations = append(opts.Run.Annotations,
		v1.ScopedLabel{
			ResourceType: v1.LabelTypeMeta,
			Key:          labels.AcornAppCuePath,
			Value:        acornCue,
		})

	app, err := rulerequest.PromptRun(ctx, opts.Client, opts.Dangerous, image, opts.Run)
	if err != nil {
		return nil, err
	}
	return app, nil
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

	_, _ = opts.Client.AppUpdate(ctx, existingApp.Name, &client.AppUpdateOptions{
		DevMode: new(bool),
	})
	return opts.Client.AppStop(ctx, existingApp.Name)
}

func runOrUpdate(ctx context.Context, acornCue, image string, opts *Options) (*apiv1.App, error) {
	_, flags, err := deployargs.ToFlagsFromImage(ctx, opts.Client, image)
	if err != nil {
		return nil, err
	}

	if len(opts.Args) > 0 {
		deployArgs, err := flags.Parse(opts.Args)
		if err != nil {
			return nil, err
		}
		opts.Run.DeployArgs = deployArgs
	}

	opts.Run.DevMode = &[]bool{true}[0]
	existingApp, err := getExistingApp(ctx, opts)
	if apierror.IsNotFound(err) {
		return createApp(ctx, acornCue, image, opts)
	} else if err != nil {
		return nil, err
	}
	return existingApp, updateApp(ctx, opts.Client, existingApp, image, opts)
}

func appDeleteStop(ctx context.Context, c client.Client, app *apiv1.App, cancel func()) error {
	w := objwatcher.New[*apiv1.App](c.GetClient())
	_, err := w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
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
	return fmt.Sprintf("STATUS: ENDPOINTS[%s] HEALTHY[%s] UPTODATE[%s] %s",
		app.Status.Columns.Endpoints,
		app.Status.Columns.Healthy,
		app.Status.Columns.UpToDate,
		app.Status.Columns.Message), app.Status.Ready
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
	w := objwatcher.New[*apiv1.App](c.GetClient())
	msg, ready := "", false
	_, err := w.ByObject(ctx, app, func(app *apiv1.App) (bool, error) {
		newMsg, newReady := appStatusMessage(app)
		if newMsg != msg || newReady != ready {
			PrintAppStatus(app)
		}
		msg, ready = newMsg, newReady
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

	opts.Run.Profiles = append([]string{"dev?"}, opts.Run.Profiles...)
	opts.Build.Profiles = append([]string{"dev?"}, opts.Build.Profiles...)

	err = buildLoop(ctx, acornCue, opts)
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
