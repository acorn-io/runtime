package dev

import (
	"context"
	"fmt"
	"sync"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/portforward"
	objwatcher "github.com/acorn-io/baaah/pkg/watcher"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/retry"
)

func DevPorts(ctx context.Context, c client.Client, appName string) error {
	wc, err := c.GetClient()
	if err != nil {
		return err
	}
	w := objwatcher.New[*apiv1.ContainerReplica](wc)

	forwarder := forwarder{
		c:                         c,
		looping:                   map[string]bool{},
		forwarding:                map[string]func(){},
		forwardingByContainerName: map[string]string{},
	}

	_, err = w.BySelector(ctx, c.GetNamespace(), nil, func(container *apiv1.ContainerReplica) (bool, error) {
		if container.Spec.AppName != appName {
			return false, nil
		}

		if container.DeletionTimestamp.IsZero() {
			forwarder.Start(ctx, container)
		} else {
			forwarder.Stop(container)
		}

		return false, nil
	})

	return err
}

type forwarder struct {
	c                         client.Client
	looping                   map[string]bool
	forwarding                map[string]func()
	forwardingByContainerName map[string]string
	mapLock                   sync.Mutex
}

func (f *forwarder) Stop(container *apiv1.ContainerReplica) {
	f.mapLock.Lock()
	defer f.mapLock.Unlock()

	cancel := f.forwarding[container.Name]
	if cancel != nil {
		cancel()
		logrus.Infof("Stopping dev ports container [%s]", container.Name)
	}

	delete(f.forwarding, container.Name)

	if f.forwardingByContainerName[container.Spec.ContainerName] == container.Name {
		delete(f.forwardingByContainerName, container.Spec.ContainerName)
	}
}

func (f *forwarder) startListener(ctx context.Context, container *apiv1.ContainerReplica, ports []v1.PortDef) {
	f.mapLock.Lock()
	defer f.mapLock.Unlock()

	if _, found := f.forwardingByContainerName[container.Spec.ContainerName]; found {
		return
	}

	if _, found := f.forwarding[container.Name]; found {
		return
	}

	ctx, cancel := context.WithCancel(ctx)

	for _, port := range ports {
		logrus.Infof("Start dev port [%s] on container [%s]", port.FormatString(""), container.Name)
		port := port
		go func() {
			if err := retry.OnError(retry.DefaultBackoff, func(err error) bool {
				return !errors.Is(err, context.Canceled)
			}, func() error {
				return portforward.PortForward(ctx, f.c, container.Name, "127.0.0.1", fmt.Sprintf("%d:%d", port.Port, port.TargetPort))
			}); err != nil && !errors.Is(err, context.Canceled) {
				f.Stop(container)
				logrus.Errorf("Failed to establish port forward for dev port [%s] on container [%s]: %v", port.FormatString(""), container.Name, err)
			}
		}()
	}

	f.forwardingByContainerName[container.Spec.ContainerName] = container.Name
	f.forwarding[container.Name] = cancel
}

func (f *forwarder) listenLoop(ctx context.Context, container *apiv1.ContainerReplica, ports []v1.PortDef) {
	defer func() {
		f.mapLock.Lock()
		defer f.mapLock.Unlock()
		delete(f.looping, container.Name)
	}()

	for {
		f.startListener(ctx, container, ports)
		select {
		case <-ctx.Done():
			break
		case <-time.After(time.Second):
		}
	}
}

func (f *forwarder) Start(ctx context.Context, container *apiv1.ContainerReplica) {
	f.mapLock.Lock()
	defer f.mapLock.Unlock()

	if f.looping[container.Name] {
		return
	}
	var ports []v1.PortDef
	for _, port := range container.Spec.Ports {
		port = port.Complete()
		if port.Dev && (port.Protocol == v1.ProtocolTCP || port.Protocol == v1.ProtocolHTTP) {
			ports = append(ports, port)
		}
	}

	if len(ports) == 0 {
		return
	}

	f.looping[container.Name] = true

	go f.listenLoop(ctx, container, ports)
}
