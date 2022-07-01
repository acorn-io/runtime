package dev

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/controller/appdefinition"
	objwatcher "github.com/acorn-io/acorn/pkg/watcher"
	"github.com/loft-sh/devspace/pkg/devspace/config/versions/latest"
	"github.com/loft-sh/devspace/pkg/devspace/sync"
	logpkg "github.com/loft-sh/devspace/pkg/util/log"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
)

func containerSyncLoop(ctx context.Context, app *apiv1.App, opts *Options) {
	go func() {
		for {
			err := containerSync(ctx, app, opts)
			if err != nil && !errors.Is(err, context.Canceled) {
				logrus.Errorf("failed to run container sync: %s", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}()
}

func containerSync(ctx context.Context, app *apiv1.App, opts *Options) error {
	syncing := map[string]bool{}
	w := objwatcher.New[*apiv1.ContainerReplica](opts.Client.GetClient())
	_, err := w.BySelector(ctx, app.Namespace, labels.Everything(), func(con *apiv1.ContainerReplica) (bool, error) {
		if con.Spec.AppName == app.Name && con.Spec.JobName == "" && con.Status.Phase == corev1.PodRunning && !syncing[con.Name] {
			for remoteDir, mount := range con.Spec.Dirs {
				if mount.ContextDir == "" {
					continue
				}
				go startSyncForPath(ctx, opts.Client, con, opts.Build.Cwd, mount.ContextDir, remoteDir, opts.BidirectionalSync)
			}
			syncing[con.Name] = true
		}
		return false, nil
	})
	return err
}

func invokeStartSyncForPath(ctx context.Context, client client.Client, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) (chan struct{}, error) {
	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		return nil, err
	}
	s, err := sync.NewSync(filepath.Join(cwd, localDir), sync.Options{
		DownstreamDisabled: !bidirectional,
		Verbose:            true,
		InitialSync:        latest.InitialSyncStrategyPreferLocal,
		Log:                logpkg.NewDefaultPrefixLogger(con.Name+" (sync) ", logpkg.GetInstance()),
	})
	if err != nil {
		return nil, err
	}

	cmd := filepath.Join(appdefinition.AcornHelperPath, strings.TrimSpace(appdefinition.AcornHelper))
	io, err := client.ContainerReplicaExec(ctx, con.Name, []string{
		cmd, "sync", "upstream", remoteDir,
	}, false, nil)
	if err != nil {
		return nil, err
	}
	if err := s.InitUpstream(io.Stdout, io.Stdin); err != nil {
		return nil, err
	}

	io, err = client.ContainerReplicaExec(ctx, con.Name, []string{
		cmd, "sync", "downstream", remoteDir,
	}, false, nil)
	if err != nil {
		return nil, err
	}
	if err := s.InitDownstream(io.Stdout, io.Stdin); err != nil {
		return nil, err
	}

	done := make(chan struct{})
	if err := s.Start(nil, nil, done, nil); err != nil {
		return nil, err
	}

	return done, nil
}

func startSyncForPath(ctx context.Context, client client.Client, con *apiv1.ContainerReplica, cwd, localDir, remoteDir string, bidirectional bool) {
	for {
		var wait <-chan struct{}
		con, err := client.ContainerReplicaGet(ctx, con.Name)
		if apierrors.IsNotFound(err) || con.Status.Phase != corev1.PodRunning {
			return
		}
		if err == nil {
			wait, err = invokeStartSyncForPath(ctx, client, con, cwd, localDir, remoteDir, bidirectional)
		}

		if err == nil {
			select {
			case <-ctx.Done():
				return
			case <-wait:
			}
		} else {
			logrus.Errorf("failed to run sync on container %s: %v", con.Name, err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
