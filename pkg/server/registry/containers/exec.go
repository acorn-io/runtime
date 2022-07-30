package containers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/watcher"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/rancher/wrangler/pkg/randomtoken"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientgo "k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultExecCmd = []string{
		"/bin/sh",
		"-c",
		"TERM=xterm-256color; export TERM; [ -x /bin/bash ] && ([ -x /usr/bin/script ] && /usr/bin/script -q -c \"/bin/bash\" /dev/null || exec /bin/bash) || exec /bin/sh",
	}
)

type ContainerExec struct {
	containers *Storage
	client     client.WithWatch
	proxy      httputil.ReverseProxy
	RESTClient clientgo.Interface
	k8s        kubernetes.Interface
}

func NewContainerExec(client client.WithWatch, containers *Storage, cfg *clientgo.Config) (*ContainerExec, error) {
	cfg = clientgo.CopyConfig(cfg)
	restconfig.SetScheme(cfg, scheme.Scheme)

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	transport, err := clientgo.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	return &ContainerExec{
		k8s:        k8s,
		client:     client,
		containers: containers,
		proxy: httputil.ReverseProxy{
			FlushInterval: 200 * time.Millisecond,
			Transport:     transport,
			Director:      func(request *http.Request) {},
		},
		RESTClient: k8s.CoreV1().RESTClient(),
	}, nil
}

func (c *ContainerExec) New() runtime.Object {
	return &apiv1.ContainerReplicaExecOptions{}
}

func (c *ContainerExec) connect(podName, podNamespace, containerName string, execOpt *apiv1.ContainerReplicaExecOptions) (http.Handler, error) {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		req := c.RESTClient.Get().
			Namespace(podNamespace).
			Resource("pods").
			Name(podName).
			SubResource("exec").
			VersionedParams(&corev1.PodExecOptions{
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       execOpt.TTY,
				Container: containerName,
				Command:   command(execOpt.Command),
			}, scheme.ParameterCodec)
		request.URL = req.URL()
		c.proxy.ServeHTTP(writer, request)
	}), nil
}

func (c *ContainerExec) Connect(ctx context.Context, id string, options runtime.Object, r rest.Responder) (http.Handler, error) {
	execOpt := options.(*apiv1.ContainerReplicaExecOptions)
	if id == "_" && execOpt.DebugImage != "" {
		return c.execNew(ctx, execOpt)
	}

	obj, err := c.containers.Get(ctx, id, nil)
	if err != nil {
		return nil, err
	}

	container := obj.(*apiv1.ContainerReplica)

	containerName := container.Spec.ContainerName
	if container.Spec.SidecarName != "" {
		containerName = container.Spec.SidecarName
	}

	if execOpt.DebugImage != "" {
		return c.execEphemeral(ctx, container, containerName, execOpt)
	}

	return c.connect(container.Status.PodName, container.Status.PodNamespace, containerName, execOpt)
}

func (c *ContainerExec) NewConnectOptions() (runtime.Object, bool, string) {
	return &apiv1.ContainerReplicaExecOptions{}, false, ""
}

func (c *ContainerExec) ConnectMethods() []string {
	return []string{"GET"}
}

func command(args []string) []string {
	if len(args) == 0 {
		return defaultExecCmd
	}
	return args
}

func (c *ContainerExec) execNew(ctx context.Context, execOpts *apiv1.ContainerReplicaExecOptions) (http.Handler, error) {
	ns, _ := request.NamespaceFrom(ctx)
	execName := "shell"
	pods := c.k8s.CoreV1().Pods(ns)
	pod, err := pods.Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "debug-shell-",
			Labels: map[string]string{
				labels.AcornDebugShell: "true",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    execName,
					Image:   execOpts.DebugImage,
					Command: []string{"sleep", "9999999"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = pods.Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
	}()

	pod, err = watcher.New[*corev1.Pod](c.client).ByObject(ctx, pod, func(pod *corev1.Pod) (bool, error) {
		for _, status := range pod.Status.ContainerStatuses {
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

	return c.connect(pod.Name, pod.Namespace, execName, execOpts)
}

func (c *ContainerExec) execEphemeral(ctx context.Context, container *apiv1.ContainerReplica, containerName string, execOpts *apiv1.ContainerReplicaExecOptions) (http.Handler, error) {
	pods := c.k8s.CoreV1().Pods(container.Status.PodNamespace)
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
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.SubPath == "" {
					volumeMounts = append(volumeMounts, volumeMount)
				}
			}
			break
		}
	}

	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            execName,
			Image:           execOpts.DebugImage,
			Command:         []string{"sleep"},
			Args:            []string{"3600"},
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			SecurityContext: nil,
			Stdin:           true,
			TTY:             execOpts.TTY,
		},
		TargetContainerName: containerName,
	})

	pod, err = pods.UpdateEphemeralContainers(ctx, pod.Name, pod, metav1.UpdateOptions{})
	if apierror.IsNotFound(err) {
		return nil, fmt.Errorf("ephemeral containers most likely unsupported by Kubernetes: %w", err)
	} else if err != nil {
		return nil, err
	}

	pod, err = watcher.New[*corev1.Pod](c.client).ByObject(ctx, pod, func(pod *corev1.Pod) (bool, error) {
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

	return c.connect(pod.Name, pod.Namespace, execName, execOpts)
}
