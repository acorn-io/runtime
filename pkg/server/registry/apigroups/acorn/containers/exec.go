package containers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/randomtoken"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/acorn-io/baaah/pkg/watcher"
	"github.com/acorn-io/mink/pkg/strategy"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/server/registry/apigroups/acorn/apps"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	defaultExecCmd = []string{
		"/bin/sh",
		"-c",
		"TERM=xterm-256color; export TERM; [ -x /bin/bash ] && ([ -x /usr/bin/script ] && /usr/bin/script -q -c \"/bin/bash\" /dev/null || exec /bin/bash) || exec /bin/sh",
	}
)

type ContainerExec struct {
	*strategy.DestroyAdapter
	client     kclient.WithWatch
	t          *Translator
	proxy      httputil.ReverseProxy
	RESTClient rest.Interface
	k8s        kubernetes.Interface
	rbac       *apps.RBACValidator
}

func NewContainerExec(client kclient.WithWatch, cfg *rest.Config) (*ContainerExec, error) {
	cfg = rest.CopyConfig(cfg)
	restconfig.SetScheme(cfg, scheme.Scheme)

	k8s, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	transport, err := rest.TransportFor(cfg)
	if err != nil {
		return nil, err
	}

	return &ContainerExec{
		k8s: k8s,
		t: &Translator{
			client: client,
		},
		client: client,
		proxy: httputil.ReverseProxy{
			FlushInterval: 200 * time.Millisecond,
			Transport:     transport,
			Director:      func(_ *http.Request) {},
		},
		RESTClient: k8s.CoreV1().RESTClient(),
		rbac:       apps.NewRBACValidator(client),
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

func (c *ContainerExec) Connect(ctx context.Context, id string, options runtime.Object, _ registryrest.Responder) (http.Handler, error) {
	execOpt := options.(*apiv1.ContainerReplicaExecOptions)

	container := &apiv1.ContainerReplica{}
	ns, _ := request.NamespaceFrom(ctx)

	err := c.client.Get(ctx, k8sclient.ObjectKey{Namespace: ns, Name: id}, container)
	if err != nil {
		return nil, err
	}

	app, err := apps.GetAppInstanceFromPublicName(ctx, c.client, ns, container.Spec.AppName)
	if err != nil {
		return nil, err
	}

	// Check spec and status permissions to ensure we don't have some race condition where
	// we validate against the wrong set of permissions
	perms := app.Spec.GrantedPermissions
	perms = append(perms, app.Spec.ImageGrantedPermissions...)
	perms = append(perms, app.Status.Permissions...)
	_, rejected, err := c.rbac.CheckPermissionsForPrivilegeEscalation(ctx, perms)
	if err != nil {
		return nil, err
	} else if len(rejected) > 0 {
		return nil, apierror.NewForbidden(schema.GroupResource{
			Group:    apiv1.SchemeGroupVersion.Group,
			Resource: "containerreplicas",
		}, id, &client.ErrNotAuthorized{
			Permissions: rejected,
		})
	}

	containerName := container.Spec.ContainerName
	if containerName == "" {
		containerName = container.Spec.JobName
	}
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
		envs         []corev1.EnvVar
		envFroms     []corev1.EnvFromSource
	)

	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if container.Name == containerName {
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.SubPath == "" {
					volumeMounts = append(volumeMounts, volumeMount)
				}
			}
			envs = container.Env
			envFroms = container.EnvFrom
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
			Env:             envs,
			EnvFrom:         envFroms,
			ImagePullPolicy: corev1.PullAlways,
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
