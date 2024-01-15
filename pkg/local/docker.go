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
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/z"
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
)

const (
	ContainerName = "acorn-local-runtime"
	volumeName    = "acorn-local-runtime-state"
)

type Container struct {
	c *client.Client
}

func NewContainer(_ context.Context) (*Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Container{
		c: cli,
	}, nil
}

func (c *Container) Ensure(ctx context.Context) (*rest.Config, error) {
	con, err := c.c.ContainerInspect(ctx, ContainerName)
	if client.IsErrNotFound(err) {
		var id string
		id, err = c.Upgrade(ctx)
		if err != nil {
			return nil, err
		}
		con, err = c.c.ContainerInspect(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	var (
		out io.ReadCloser
	)
	for i := 0; ; i++ {
		if i > 20 {
			return nil, fmt.Errorf("timeout trying to launch %s container", ContainerName)
		}
		out, _, err = c.c.CopyFromContainer(ctx, con.ID, "/etc/rancher/k3s/k3s.yaml")
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
	cfg.Host = fmt.Sprintf("https://localhost:%s", con.NetworkSettings.Ports["6443/tcp"][0].HostPort)

	restconfig.SetScheme(cfg, scheme.Scheme)
	return cfg, waitFor(ctx, cfg)
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
					Force: true,
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
		RemoveLinks:   false,
		Force:         true,
	})
	if client.IsErrNotFound(err) {
	} else if err != nil {
		return err
	}

	if data {
		if err := c.c.VolumeRemove(ctx, volumeName, false); client.IsErrNotFound(err) {
		} else if err != nil {
			return err
		}
	}

	return c.DeletePorts(ctx)
}

func (c *Container) Upgrade(ctx context.Context) (string, error) {
	con, err := c.c.ContainerInspect(ctx, ContainerName)
	if client.IsErrNotFound(err) {
		return c.Create(ctx, false)
	} else if err != nil {
		return "", err
	}

	if con.Config.Image == system.DefaultImage() {
		return con.ID, c.Start(ctx)
	}

	if err := c.Delete(ctx, false); err != nil {
		return "", err
	}

	return c.Create(ctx, false)
}

func (c *Container) Reset(ctx context.Context) error {
	if err := c.Delete(ctx, true); err != nil {
		return err
	}
	_, err := c.Create(ctx, false)
	return err
}

func (c *Container) Create(ctx context.Context, upgrade bool) (string, error) {
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
					HostPort: "",
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
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
		},
	}, nil, nil, ContainerName)
	if client.IsErrNotFound(err) {
		resp, err := c.c.ImagePull(ctx, system.DefaultImage(), types.ImagePullOptions{})
		if err != nil {
			return "", err
		}
		out := streams.NewOut(os.Stdout)
		if err := jsonmessage.DisplayJSONMessagesToStream(resp, out, nil); err != nil {
			return "", err
		}
		return c.Create(ctx, false)
	} else if errdefs.IsConflict(err) {
		if upgrade {
			return c.Upgrade(ctx)
		} else {
			return con.ID, c.Start(ctx)
		}
	} else if err != nil {
		return "", err
	}

	return con.ID, c.Start(ctx)
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
	if client.IsErrNotFound(err) {
	} else if err != nil {
		return err
	}
	return c.DeletePorts(ctx)
}
