package local

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/watcher"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/install"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/runtime/pkg/term"
	"github.com/acorn-io/z"
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ContainerName = "acorn-local-runtime"
	volumeName    = "acorn-local-runtime-state"
)

type Container struct {
	c client.APIClient
}

func NewContainer() (*Container, error) {
	cli, err := command.NewDockerCli()
	if err != nil {
		return nil, err
	}
	if err := cli.Initialize(&cliflags.ClientOptions{}); err != nil {
		return nil, err
	}
	return &Container{
		c: cli.Client(),
	}, nil
}

func (c *Container) getKubeconfig(ctx context.Context, port string) (*rest.Config, error) {
	var (
		err error
		out io.ReadCloser
	)
	for i := 0; ; i++ {
		if i > 20 {
			return nil, fmt.Errorf("timeout trying to launch %s container", ContainerName)
		}
		out, _, err = c.c.CopyFromContainer(ctx, ContainerName, "/etc/rancher/k3s/k3s.yaml")
		if client.IsErrNotFound(err) {
			time.Sleep(500 * time.Millisecond)
			continue
		} else if err != nil {
			return nil, err
		}
		break
	}
	defer out.Close()

	tar := tar.NewReader(out)
	_, err = tar.Next()
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(tar)
	if err != nil {
		return nil, err
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(data)
	if err != nil {
		return nil, err
	}
	cfg.Host = fmt.Sprintf("https://localhost:%s", port)

	restconfig.SetScheme(cfg, scheme.Scheme)
	return cfg, waitFor(ctx, cfg)
}

func (c *Container) Ensure(ctx context.Context) (*rest.Config, error) {
	_, port, err := c.Upgrade(ctx, true)
	if err != nil {
		return nil, err
	}

	return c.getKubeconfig(ctx, port)
}

func waitFor(ctx context.Context, cfg *rest.Config) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cli, err := rest.UnversionedRESTClientFor(cfg)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// Return close error along with the last ready error, if any
			return errors.Join(fmt.Errorf("context closed before %q was ready: %w", cfg.Host, ctx.Err()), err)
		default:
		}

		resp := cli.Get().AbsPath("/readyz").Do(ctx)
		if err = resp.Error(); err == nil {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func (c *Container) DeletePorts(ctx context.Context) error {
	cons, err := c.c.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return err
	}

	for _, con := range cons {
		for _, name := range con.Names {
			if strings.HasPrefix(name, "/"+ContainerName+"-") {
				if err := c.c.ContainerRemove(ctx, name, types.ContainerRemoveOptions{
					RemoveVolumes: true,
					Force:         true,
				}); err != nil {
					return err
				}
				break
			}
		}
	}

	return nil
}

func (c *Container) Delete(ctx context.Context, data bool) error {
	err := c.c.ContainerRemove(ctx, ContainerName, types.ContainerRemoveOptions{
		RemoveVolumes: data,
		Force:         true,
	})
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}

	if data {
		if err := c.c.VolumeRemove(ctx, volumeName, false); err != nil && !client.IsErrNotFound(err) {
			return err
		}
	}

	return c.DeletePorts(ctx)
}

func (c *Container) Upgrade(ctx context.Context, ignoreLocal bool) (string, string, error) {
	con, err := c.c.ContainerInspect(ctx, ContainerName)
	if client.IsErrNotFound(err) {
		if _, err := c.Create(ctx); err != nil {
			return "", "", err
		}
		con, err = c.c.ContainerInspect(ctx, ContainerName)
		if err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}

	if con.State == nil || !con.State.Running {
		if err := c.Start(ctx); err != nil {
			return "", "", err
		}
		if err := c.Wait(ctx); err != nil {
			return "", "", err
		}
		con, err = c.c.ContainerInspect(ctx, ContainerName)
		if err != nil {
			return "", "", err
		}
	}

	if con.Config.Image == system.DefaultImage() || (ignoreLocal && con.Config.Image == "localdev") {
		return con.ID, con.NetworkSettings.Ports["6443/tcp"][0].HostPort, c.Start(ctx)
	}

	if err := c.Delete(ctx, false); err != nil {
		return "", "", err
	}

	return c.Upgrade(ctx, ignoreLocal)
}

func (c *Container) Reset(ctx context.Context, data bool) error {
	if err := c.Delete(ctx, data); err != nil {
		return err
	}
	_, err := c.Create(ctx)
	return err
}

func (c *Container) Wait(ctx context.Context) error {
	pb := &term.Builder{}

	imageStatus := pb.New("Image pulled")
	imageStatus.Infof("Pulling image %s", system.DefaultImage())
	if err := c.pull(ctx); err != nil {
		return imageStatus.Fail(err)
	}
	imageStatus.Success()

	conStatus := pb.New("Container created (to delete \"acorn local rm\")")
	conStatus.Infof("Creating")

	for {
		_, err := c.c.ContainerInspect(ctx, ContainerName)
		if err == nil {
			conStatus.Success()
			break
		} else if err != nil && !client.IsErrNotFound(err) {
			return conStatus.Fail(err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	running := pb.New("Container running (to stop \"acorn local stop\")")
	running.Infof("Starting")

	var port string

	for {
		con, err := c.c.ContainerInspect(ctx, ContainerName)
		if err != nil {
			return running.Fail(err)
		}

		if con.State == nil || !con.State.Running {
			select {
			case <-ctx.Done():
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		port = con.NetworkSettings.Ports["6443/tcp"][0].HostPort
		break
	}

	running.Success()

	restConfig, err := c.getKubeconfig(ctx, port)
	if err != nil {
		return err
	}

	kc, err := kclient.NewWithWatch(restConfig, kclient.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return err
	}

	if err := install.WaitAPI(ctx, pb, 1, system.LocalImageBind, kc); err != nil {
		return err
	}

	ns := pb.New("Local project created")
	ns.Infof("Waiting for local project")
	w := watcher.New[*v1.Project](kc)
	for {
		_, err = w.ByName(ctx, "", "local", func(obj *v1.Project) (bool, error) {
			return true, nil
		})
		if err != nil {
			ns.Infof("Waiting for local project: %v", err)
			select {
			case <-ctx.Done():
				return ns.Fail(ctx.Err())
			case <-time.After(1 * time.Second):
			}
			continue
		}
		break
	}
	ns.Success()
	return nil
}

func (c *Container) pull(ctx context.Context) error {
	_, _, err := c.c.ImageInspectWithRaw(ctx, system.DefaultImage())
	if err == nil {
		return nil
	}

	resp, err := c.c.ImagePull(ctx, system.DefaultImage(), types.ImagePullOptions{})
	if err != nil {
		return err
	}
	out := streams.NewOut(os.Stdout)
	return jsonmessage.DisplayJSONMessagesToStream(resp, out, nil)
}

func (c *Container) Create(ctx context.Context) (string, error) {
	v, err := c.c.VolumeCreate(ctx, volume.CreateOptions{
		Name: volumeName,
	})
	if err != nil {
		return "", err
	}

	con, err := c.c.ContainerCreate(ctx, &container.Config{
		Cmd: []string{
			"local", "server",
		},
		Env: []string{
			"ACORN_DOCKER_IMAGE=" + system.DefaultImage(),
		},
		Image: system.DefaultImage(),
		Volumes: map[string]struct{}{
			"/var/lib/rancher/k3s": {},
			"/var/run/docker.sock": {},
		},
		ExposedPorts: map[nat.Port]struct{}{
			"6443/tcp": {},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			"6443/tcp": {
				{
					HostIP:   "0.0.0.0",
					HostPort: os.Getenv("ACORN_LOCAL_PORT"),
				},
			},
		},
		Privileged: true,
		Tmpfs: map[string]string{
			"/run": "",
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: v.Name,
				Target: "/var/lib/rancher/k3s",
			},
			{
				Type:   mount.TypeVolume,
				Source: v.Name,
				Target: "/var/lib/buildkit",
			},
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
			{
				Type:     mount.TypeBind,
				Source:   "/lib/modules",
				Target:   "/lib/modules",
				ReadOnly: true,
			},
		},
	}, nil, nil, ContainerName)
	if client.IsErrNotFound(err) {
		if err := c.pull(ctx); err != nil {
			return "", err
		}
		return c.Create(ctx)
	} else if errdefs.IsConflict(err) {
		return con.ID, c.Start(ctx)
	} else if err != nil {
		return "", err
	}

	if err := c.Start(ctx); err != nil {
		return "", err
	}

	return con.ID, c.Wait(ctx)
}

func (c *Container) Start(ctx context.Context) error {
	return c.c.ContainerStart(ctx, ContainerName, types.ContainerStartOptions{})
}

type LogOptions struct {
	Tail   string `usage:"Number of lines to show from the end of the logs" default:"all" short:"n"`
	Follow bool   `usage:"Follow log output" short:"f"`
}

func (c *Container) Logs(ctx context.Context, opt LogOptions) error {
	logs, err := c.c.ContainerLogs(ctx, ContainerName, types.ContainerLogsOptions{
		ShowStderr: true,
		ShowStdout: true,
		Follow:     opt.Follow,
		Tail:       opt.Tail,
	})
	if err != nil {
		return err
	}
	defer logs.Close()

	_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	return err
}

func (c *Container) Stop(ctx context.Context) error {
	err := c.c.ContainerStop(ctx, ContainerName, container.StopOptions{
		Timeout: z.Pointer(5),
	})
	if err != nil && !client.IsErrNotFound(err) {
		return err
	}
	return c.DeletePorts(ctx)
}
