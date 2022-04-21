package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/rancher/wrangler/pkg/randomtoken"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	defaultExecCmd = []string{
		"/bin/sh",
		"-c",
		"TERM=xterm-256color; export TERM; [ -x /bin/bash ] && ([ -x /usr/bin/script ] && /usr/bin/script -q -c \"/bin/bash\" /dev/null || exec /bin/bash) || exec /bin/sh",
	}
)

func (c *client) execEphemeral(ctx context.Context, container *ContainerReplica, args []string, tty bool, containerName, image string) (*term.ExecIO, error) {
	k8s, err := kubernetes.NewForConfig(c.RESTConfig)
	if err != nil {
		return nil, err
	}

	pods := k8s.CoreV1().Pods(container.Status.PodNamespace)
	pod, err := pods.Get(ctx, container.Status.PodName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	unique, err := randomtoken.Generate()
	if err != nil {
		return nil, err
	}

	var (
		execName     = name.SafeConcatName(containerName, "exec", unique[:8])
		volumeMounts []corev1.VolumeMount
	)

	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if container.Name == containerName {
			volumeMounts = container.VolumeMounts
			break
		}
	}

	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            execName,
			Image:           image,
			Command:         []string{"sleep"},
			Args:            []string{"3600"},
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: nil,
			Stdin:           true,
			TTY:             tty,
		},
		TargetContainerName: containerName,
	})

	pod, err = pods.UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{})
	if apierror.IsNotFound(err) {
		return nil, fmt.Errorf("ephemeral containers most likely unsupported by Kubernetes: %w", err)
	} else if err != nil {
		return nil, err
	}

	messageCtx, messageCancel := context.WithCancel(ctx)
	defer messageCancel()
	go func() {
		select {
		case <-messageCtx.Done():
		case <-time.After(10 * time.Second):
			fmt.Printf("Waiting for ephemeral container %s/%s for image %s to start\n", pod.Name, execName, image)
		}
	}()

	_, err = watcher.New[*corev1.Pod](c.Client).ByObject(ctx, pod, func(pod *corev1.Pod) (bool, error) {
		for _, status := range pod.Status.EphemeralContainerStatuses {
			if status.Name == execName {
				if status.State.Running != nil {
					return true, nil
				} else if status.State.Terminated != nil {
					return false, fmt.Errorf("%s: %s", status.State.Terminated.Reason, status.State.Terminated.Message)
				}
			}
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	messageCancel()

	exec, err := c.execContainerForName(ctx, pod.Name, pod.Namespace, execName, args, tty)
	if err != nil {
		return nil, err
	}

	oldExit := exec.ExitCode
	newExit := make(chan term.ExitCode, 1)
	go func() {
		result := <-oldExit

		pod, err := pods.Get(ctx, container.Status.PodName, metav1.GetOptions{})
		if err != nil {
			return
		}

		var newEphemeralContainers []corev1.EphemeralContainer
		for _, e := range pod.Spec.EphemeralContainers {
			if e.Name == execName {
				continue
			}
			newEphemeralContainers = append(newEphemeralContainers, e)
		}
		pod.Spec.EphemeralContainers = newEphemeralContainers

		// This doesn't actually work, seems like we aren't allowed to delete ephemeral containers
		pods.UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{})
		newExit <- result
		close(newExit)
	}()

	exec.ExitCode = newExit
	return exec, nil
}

func (c *client) execContainer(ctx context.Context, container *ContainerReplica, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	containerName := container.ContainerName
	if container.SidecarName != "" {
		containerName = container.SidecarName
	}

	if opts != nil && opts.DebugImage != "" {
		return c.execEphemeral(ctx, container, args, tty, containerName, opts.DebugImage)
	}

	return c.execContainerForName(ctx, container.Status.PodName, container.Status.PodNamespace, containerName, args, tty)
}

func (c *client) execContainerForName(ctx context.Context, podName, podNamespace, containerName string, args []string, tty bool) (*term.ExecIO, error) {
	req := c.RESTClient.Get().
		Namespace(podNamespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       tty,
			Container: containerName,
			Command:   command(args),
		}, scheme.ParameterCodec)

	conn, err := c.Dialer.DialContext(ctx, req.URL().String(), nil)
	if err != nil {
		return nil, err
	}

	exit := make(chan term.ExitCode, 1)
	go func() {
		exit <- term.ToExitCode(conn.ForStream(3))
	}()

	resize := make(chan term.TermSize, 1)
	go func() {
		for size := range resize {
			data, err := json.Marshal(size)
			if err != nil {
				logrus.Errorf("failed to marshall term size %v: %v", size, err)
				continue
			}
			_, err = conn.Write(4, data)
			if err != nil {
				break
			}
		}
	}()

	return &term.ExecIO{
		Stdin:    conn.ForStream(0),
		Stdout:   conn.ForStream(1),
		Stderr:   conn.ForStream(2),
		ExitCode: exit,
		Resize:   resize,
	}, nil
}

func command(args []string) []string {
	if len(args) == 0 {
		return defaultExecCmd
	}
	return args
}

func (c *client) ContainerReplicaExec(ctx context.Context, containerName string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error) {
	con, err := c.ContainerReplicaGet(ctx, containerName)
	if err != nil {
		return nil, err
	}

	return c.execContainer(ctx, con, args, tty, opts)
}
