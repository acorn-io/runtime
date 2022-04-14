package client

import (
	"context"

	"github.com/ibuildthecloud/baaah/pkg/restconfig"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/client/term"
	"github.com/ibuildthecloud/herd/pkg/k8schannel"
	"github.com/ibuildthecloud/herd/pkg/k8sclient"
	"github.com/ibuildthecloud/herd/pkg/system"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type App struct {
	Name        string            `json:"name,omitempty"`
	Created     metav1.Time       `json:"created,omitempty"`
	Revision    string            `json:"revision,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Image   string             `json:"image,omitempty"`
	Volumes []v1.VolumeBinding `json:"volumes,omitempty"`
	Secrets []v1.SecretBinding `json:"secrets,omitempty"`

	Status v1.AppInstanceStatus `json:"status,omitempty"`
}

type ContainerReplica struct {
	Name          string            `json:"name,omitempty"`
	AppName       string            `json:"appName,omitempty"`
	JobName       string            `json:"jobName,omitempty"`
	ContainerName string            `json:"containerName,omitempty"`
	SidecarName   string            `json:"sidecarName,omitempty"`
	Created       metav1.Time       `json:"created,omitempty"`
	Revision      string            `json:"revision,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`

	Dirs        map[string]v1.VolumeMount `json:"dirs,omitempty"`
	Files       map[string]v1.File        `json:"files,omitempty"`
	Image       string                    `json:"image,omitempty"`
	Build       *v1.Build                 `json:"build,omitempty"`
	Command     []string                  `json:"command,omitempty"`
	Interactive bool                      `json:"interactive,omitempty"`
	Entrypoint  []string                  `json:"entrypoint,omitempty"`
	Environment []v1.EnvVar               `json:"environment,omitempty"`
	WorkingDir  string                    `json:"workingDir,omitempty"`
	Ports       []v1.Port                 `json:"ports,omitempty"`

	// Init is only available on sidecars
	Init bool `json:"init,omitempty"`

	// Sidecars are not available on sidecars
	Sidecars map[string]v1.Container `json:"sidecars,omitempty"`

	Status ContainerReplicaStatus `json:"status,omitempty"`
}

type ContainerReplicaColumns struct {
	State string `json:"state,omitempty"`
	App   string `json:"app,omitempty"`
}

type ContainerReplicaStatus struct {
	PodName      string          `json:"podName,omitempty"`
	PodNamespace string          `json:"podNamespace,omitempty"`
	Phase        corev1.PodPhase `json:"phase,omitempty"`
	PodMessage   string          `json:"message,omitempty"`
	PodReason    string          `json:"reason,omitempty"`

	Columns              ContainerReplicaColumns `json:"columns,omitempty"`
	State                corev1.ContainerState   `json:"state,omitempty"`
	LastTerminationState corev1.ContainerState   `json:"lastState,omitempty"`
	Ready                bool                    `json:"ready"`
	RestartCount         int32                   `json:"restartCount"`
	Image                string                  `json:"image"`
	ImageID              string                  `json:"imageID"`
	Started              *bool                   `json:"started,omitempty"`
}

func Default() (Client, error) {
	cfg, err := restconfig.Default()
	if err != nil {
		return nil, err
	}

	ns := system.UserNamespace()
	return New(cfg, ns)
}

func New(restconfig *rest.Config, namespace string) (Client, error) {
	k8sclient, err := k8sclient.New(restconfig)
	if err != nil {
		return nil, err
	}

	dialer, err := k8schannel.NewDialer(restconfig, false)
	if err != nil {
		return nil, err
	}

	cfg := rest.CopyConfig(restconfig)
	cfg.APIPath = "/api"
	cfg.GroupVersion = &schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}

	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	return &client{
		Namespace:  namespace,
		Client:     k8sclient,
		RESTConfig: restconfig,
		RESTClient: restClient,
		Dialer:     dialer,
	}, nil
}

type Client interface {
	AppList(ctx context.Context) ([]App, error)
	AppDelete(ctx context.Context, name string) error
	AppGet(ctx context.Context, name string) (*App, error)
	AppStop(ctx context.Context, name string) error
	AppStart(ctx context.Context, name string) error

	ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]ContainerReplica, error)
	ContainerReplicaGet(ctx context.Context, name string) (*ContainerReplica, error)
	ContainerReplicaDelete(ctx context.Context, name string) error
	ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error)

	VolumeCreate(ctx context.Context, name string, capacity resource.Quantity, opts *VolumeCreateOptions) (*Volume, error)
	VolumeList(ctx context.Context) ([]Volume, error)
	VolumeGet(ctx context.Context, name string) (*Volume, error)
	VolumeDelete(ctx context.Context, name string) error
}

type VolumeCreateOptions struct {
	AccessModes []v1.AccessMode `json:"accessModes,omitempty"`
	Class       string          `json:"class,omitempty"`
}

type Volume struct {
	Name        string            `json:"name,omitempty"`
	Created     metav1.Time       `json:"created,omitempty"`
	Revision    string            `json:"revision,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`

	Capacity    *resource.Quantity `json:"capacity,omitempty"`
	AccessModes []v1.AccessMode    `json:"accessModes,omitempty"`
	Class       string             `json:"class,omitempty"`
	Status      VolumeStatus       `json:"status,omitempty"`
}

type VolumeStatus struct {
	AppName      string        `json:"appName,omitempty"`
	AppNamespace string        `json:"appNamespace,omitempty"`
	VolumeName   string        `json:"volumeName,omitempty"`
	Status       string        `json:"status,omitempty"`
	Reason       string        `json:"reason,omitempty"`
	Message      string        `json:"message,omitempty"`
	Columns      VolumeColumns `json:"columns,omitempty"`
}

type VolumeColumns struct {
	AccessModes string `json:"accessModes,omitempty"`
}

type ContainerReplicaExecOptions struct {
	DebugImage string `json:"debugImage,omitempty"`
}

type ContainerReplicaListOptions struct {
	App string `json:"app,omitempty"`
}

func (c *ContainerReplicaListOptions) complete() *ContainerReplicaListOptions {
	if c == nil {
		return &ContainerReplicaListOptions{}
	}
	return c
}

type client struct {
	Namespace  string `json:"namespace,omitempty"`
	Client     kclient.WithWatch
	RESTConfig *rest.Config
	RESTClient *rest.RESTClient
	Dialer     *k8schannel.Dialer
}

func waitAndClose[T any](eg *errgroup.Group, c chan T, err *error) {
	go func() {
		*err = eg.Wait()
		close(c)
	}()
}

func less(terms ...string) bool {
	for i := range terms {
		if i%2 != 0 {
			continue
		}
		if terms[i] == terms[i+1] {
			continue
		}
		return terms[i] < terms[i+1]
	}
	return false
}
