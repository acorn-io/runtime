package dev

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	sync2 "sync"
	"time"

	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/controller/appdefinition"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/moby/buildkit/frontend/dockerfile/dockerignore"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func containerSyncLoop(ctx context.Context, client client.Client, logger Logger, appName string, watcher *watcher, opts *Options) error {
	for {
		err := containerSync(ctx, client, logger, appName, watcher, opts)
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

func containerSync(ctx context.Context, client client.Client, logger Logger, appName string, watcher *watcher, opts *Options) error {
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
	_, err = w.BySelector(ctx, client.GetNamespace(), labels.Everything(), func(con *apiv1.ContainerReplica) (bool, error) {
		if con.Spec.AppName == appName && con.Spec.JobName == "" && con.Status.Phase == corev1.PodRunning && !syncing[con.Name] {
			if con.Spec.Init {
				return false, nil
			}

			watcher.addWatchFiles(con.Spec.Build.WatchFiles...)

			for remoteDir, mount := range con.Spec.Dirs {
				if mount.ContextDir == "" {
					continue
				}
				var (
					remoteDir = remoteDir
					mount     = mount
				)
				go func() {
					startSyncForPath(ctx, client, logger, con, cwd, mount.ContextDir, remoteDir, opts.BidirectionalSync)
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

func findDockerIgnore(path string) (string, error) {
	startPath := filepath.Join(path, ".dockerignore")
	for {
		testPath := filepath.Join(path, ".dockerignore")
		if _, err := os.Stat(testPath); err == nil {
			return testPath, nil
		} else if errors.Is(err, fs.ErrNotExist) {
			newPath := filepath.Dir(path)
			if newPath == path {
				return startPath, nil
			}
			if _, err := os.Stat(newPath); errors.Is(err, fs.ErrNotExist) {
				return startPath, nil
			} else if err != nil {
				return "", err
			}
			path = newPath
		} else {
			return "", err
		}
	}
}

func invokeStartSyncForPath(ctx context.Context, client client.Client, logger Logger, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) (chan struct{}, chan error, error) {
	source := filepath.Join(cwd, localDir)
	if s, err := os.Stat(source); err == nil && !s.IsDir() {
		return nil, nil, nil
	}
	err := os.MkdirAll(source, 0755)
	if err != nil {
		return nil, nil, err
	}
	var exclude []string
	dockerIgnorePath, err := findDockerIgnore(filepath.Join(cwd, localDir))
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(dockerIgnorePath)
	if err == nil {
		lines, err := dockerignore.ReadAll(f)
		_ = f.Close()
		if err == nil {
			exclude = lines
		} else {
			logrus.Warnf("failed to read %s for syncing: %v", filepath.Join(cwd, ".dockerignore"), err)
			exclude = nil
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		logrus.Warnf("failed to open %s for syncing: %v", filepath.Join(cwd, ".dockerignore"), err)
		exclude = nil
	}
	s, err := sync.NewSync(ctx, source, sync.Options{
		DownstreamDisabled: !bidirectional,
		Polling:            true,
		Verbose:            true,
		UploadExcludePaths: exclude,
		InitialSync:        latest.InitialSyncStrategyPreferLocal,
		Log: newLogger(logger, con).
			WithPrefix("(sync): "),
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

type containerLogWriter struct {
	logger Logger
	con    *apiv1.ContainerReplica
}

func (c containerLogWriter) Write(p []byte) (n int, err error) {
	c.logger.Container(metav1.Now(), c.con.Name, strings.TrimSpace(string(p)))
	return len(p), nil
}

func newLogger(logger Logger, con *apiv1.ContainerReplica) logpkg.Logger {
	out := containerLogWriter{
		logger: logger,
		con:    con,
	}
	return logpkg.NewStreamLoggerWithFormat(&ignore{
		Out: out,
	}, &ignore{
		Out: out,
	}, logrus.GetLevel(), logpkg.RawFormat)
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

func startSyncForPath(ctx context.Context, client client.Client, logger Logger, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) {
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
			wait, waiterr, err = invokeStartSyncForPath(ctx, client, logger, con, cwd, localDir, remoteDir, bidirectional)
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
