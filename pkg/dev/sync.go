package dev

import (
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	sync2 "sync"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
)

func containerSyncLoop(ctx context.Context, client client.Client, app *apiv1.App, opts *Options) error {
	for {
		err := containerSync(ctx, client, app, opts)
		if err != nil && !errors.Is(err, context.Canceled) {
			logrus.Errorf("failed to run container sync: %s", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func containerSync(ctx context.Context, client client.Client, app *apiv1.App, opts *Options) error {
	cwd, file, err := opts.ImageSource.ResolveImageAndFile()
	if err != nil {
		return err
	}

	if file == "" {
		// not a built image, no sync
		return nil
	}

	syncLock := sync2.Mutex{}
	syncing := map[string]bool{}
	wc, err := client.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.ContainerReplica](wc)
	_, err = w.BySelector(ctx, app.Namespace, labels.Everything(), func(con *apiv1.ContainerReplica) (bool, error) {
		if con.Spec.AppName == app.Name && con.Spec.JobName == "" && con.Status.Phase == corev1.PodRunning && !syncing[con.Name] {
			if con.Spec.Init {
				return false, nil
			}
			for remoteDir, mount := range con.Spec.Dirs {
				if mount.ContextDir == "" {
					continue
				}
				var (
					remoteDir = remoteDir
					mount     = mount
				)
				go func() {
					startSyncForPath(ctx, client, con, cwd, mount.ContextDir, remoteDir, opts.BidirectionalSync)
					syncLock.Lock()
					delete(syncing, con.Name)
					syncLock.Unlock()
				}()
			}
			syncLock.Lock()
			syncing[con.Name] = true
			syncLock.Unlock()
		}
		return false, nil
	})
	return err
}

func invokeStartSyncForPath(ctx context.Context, client client.Client, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) (chan struct{}, chan error, error) {
	source := filepath.Join(cwd, localDir)
	if s, err := os.Stat(source); err == nil && !s.IsDir() {
		return nil, nil, nil
	}
	err := os.MkdirAll(source, 0755)
	if err != nil {
		return nil, nil, err
	}
	var exclude []string
	f, err := os.Open(filepath.Join(cwd, ".dockerignore"))
	if err == nil {
		lines, err := dockerignore.ReadAll(f)
		_ = f.Close()
		if err == nil {
			exclude = lines
		} else {
			logrus.Warnf("failed to read %s for syncing: %v", filepath.Join(cwd, ".dockerignore"), err)
			exclude = nil
		}
	} else if !os.IsNotExist(err) {
		logrus.Warnf("failed to open %s for syncing: %v", filepath.Join(cwd, ".dockerignore"), err)
		exclude = nil
	}
	s, err := sync.NewSync(ctx, source, sync.Options{
		DownstreamDisabled: !bidirectional,
		Polling:            true,
		Verbose:            true,
		UploadExcludePaths: exclude,
		InitialSync:        latest.InitialSyncStrategyPreferLocal,
		Log: newLogger().
			WithPrefix(strings.TrimPrefix(con.Name, con.Spec.AppName+".") + ": (sync): "),
	})
	if err != nil {
		return nil, nil, err
	}

	cmd := path.Join(appdefinition.AcornHelperPath, strings.TrimSpace(appdefinition.AcornHelper))
	io, err := client.ContainerReplicaExec(ctx, con.Name, []string{
		cmd, "sync", "upstream", remoteDir,
	}, false, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.InitUpstream(io.Stdout, io.Stdin); err != nil {
		return nil, nil, err
	}

	io, err = client.ContainerReplicaExec(ctx, con.Name, []string{
		cmd, "sync", "downstream", remoteDir,
	}, false, nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.InitDownstream(io.Stdout, io.Stdin); err != nil {
		return nil, nil, err
	}

	done := make(chan struct{})
	waiterr := make(chan error, 1)
	if err := s.Start(nil, nil, done, waiterr); err != nil {
		return nil, nil, err
	}

	return done, waiterr, nil
}

func newLogger() logpkg.Logger {
	return logpkg.NewStreamLogger(&ignore{
		Out: os.Stdout,
	}, &ignore{
		Out: os.Stderr,
	}, logrus.InfoLevel)
}

type ignore struct {
	Out io.Writer
}

func (i *ignore) Write(p []byte) (n int, err error) {
	if !strings.Contains(string(p), "(sync)") {
		return len(p), nil
	}
	return i.Out.Write(p)
}

func startSyncForPath(ctx context.Context, client client.Client, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) {
	for {
		var (
			wait    <-chan struct{}
			waiterr <-chan error
		)

		con, err := client.ContainerReplicaGet(ctx, con.Name)
		if apierrors.IsNotFound(err) || con.Status.Phase != corev1.PodRunning {
			return
		}
		if err == nil {
			wait, waiterr, err = invokeStartSyncForPath(ctx, client, con, cwd, localDir, remoteDir, bidirectional)
		}

		if err == nil {
			select {
			case <-ctx.Done():
				return
			case <-wait:
			case <-waiterr:
			}
		} else {
			logrus.Debugf("failed to run sync on container %s: %v", con.Name, err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
