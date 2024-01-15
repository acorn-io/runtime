package local

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/local"
	"github.com/acorn-io/runtime/pkg/local/webhook"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Handler struct {
	c            *client.Client
	targetIP     string
	targetIPLock sync.Mutex
}

func NewHandler() (*Handler, error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Handler{
		c: c,
	}, nil
}

func (c *Handler) ProvisionPorts(req router.Request, resp router.Response) error {
	svc := req.Object.(*corev1.Service)
	if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return nil
	}

	if err := c.getIP(req.Ctx); err != nil {
		return err
	}

	for _, port := range svc.Spec.Ports {
		if port.Port == 0 {
			continue
		}
		name := strings.ToLower(fmt.Sprintf("%s-%d-%s", local.ContainerName, port.Port, port.Protocol))
		if err := c.ensure(req.Ctx, name, port.Port, string(port.Protocol)); err != nil {
			return err
		}

		if svc.Spec.ClusterIP == "" {
			continue
		}

		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: system.Namespace,
				Labels: map[string]string{
					"app": "klipper-lb",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "klipper-lb",
						Command: []string{"klipper-lb"},
						Image:   system.LocalImage,
						SecurityContext: &corev1.SecurityContext{
							Capabilities: &corev1.Capabilities{
								Add: []corev1.Capability{
									"NET_ADMIN",
								},
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  "SRC_PORT",
								Value: fmt.Sprint(port.Port),
							},
							{
								Name:  "SRC_RANGES",
								Value: "0.0.0.0/0",
							},
							{
								Name:  "DEST_PROTO",
								Value: string(port.Protocol),
							},
							{
								Name:  "DEST_PORT",
								Value: fmt.Sprint(port.Port),
							},
							{
								Name:  "DEST_IPS",
								Value: svc.Spec.ClusterIP,
							},
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          "port",
								HostPort:      port.Port,
								ContainerPort: port.Port,
								Protocol:      port.Protocol,
							},
						},
					},
				},
			},
		}

		// Patch inline now, otherwise baaah will fight with the changes the webhook makes
		webhook.PatchPodSpec(&pod.Spec)
		resp.Objects(pod)

		svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
			{
				Hostname: "localhost",
			},
		}
	}

	return nil
}

func (c *Handler) create(ctx context.Context, name string, port int32, proto string) error {
	_, err := c.c.ContainerCreate(ctx, &container.Config{
		Env: []string{
			fmt.Sprintf("SRC_PORT=%d", port),
			"SRC_RANGES=0.0.0.0/0",
			fmt.Sprintf("DEST_PROTO=%s", proto),
			fmt.Sprintf("DEST_PORT=%d", port),
			fmt.Sprintf("DEST_IPS=%s", c.targetIP),
		},
		Image:      system.LocalDockerImage,
		Entrypoint: []string{"klipper-lb"},
	}, &container.HostConfig{
		Privileged: true,
		PortBindings: map[nat.Port][]nat.PortBinding{
			nat.Port(strings.ToLower(fmt.Sprintf("%d/%s", port, proto))): {
				{
					HostPort: fmt.Sprint(port),
				},
			},
		},
	}, nil, nil, name)
	if err != nil {
		return err
	}
	return c.c.ContainerStart(ctx, name, types.ContainerStartOptions{})
}

func wrongTarget(con types.ContainerJSON, targetIP string) bool {
	targetIPEnv := "DEST_IPS=" + targetIP
	for _, env := range con.Config.Env {
		if env == targetIPEnv {
			return false
		}
	}
	return true
}

func (c *Handler) ensure(ctx context.Context, name string, port int32, proto string) error {
	con, err := c.c.ContainerInspect(ctx, name)
	if client.IsErrNotFound(err) {
		return c.create(ctx, name, port, proto)
	} else if err != nil {
		return err
	}

	if con.Config.Image != system.LocalDockerImage || wrongTarget(con, c.targetIP) {
		err = c.c.ContainerRemove(ctx, name, types.ContainerRemoveOptions{
			Force: true,
		})
		if err != nil {
			return err
		}
		return c.create(ctx, name, port, proto)
	}

	return c.c.ContainerStart(ctx, name, types.ContainerStartOptions{})
}

func (c *Handler) getIP(ctx context.Context) error {
	c.targetIPLock.Lock()
	defer c.targetIPLock.Unlock()

	if c.targetIP != "" {
		return nil
	}

	con, err := c.c.ContainerInspect(ctx, local.ContainerName)
	if err != nil {
		return err
	}

	c.targetIP = con.NetworkSettings.IPAddress
	return nil
}
