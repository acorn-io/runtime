package client

import (
	"context"
	"net"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client/term"
	"github.com/acorn-io/acorn/pkg/k8schannel"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"k8s.io/client-go/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Default() (Client, error) {
	cfg, err := restconfig.Default()
	if err != nil {
		return nil, err
	}

	ns := system.UserNamespace()
	return New(cfg, ns)
}

func New(restConfig *rest.Config, namespace string) (Client, error) {
	if namespace == "" {
		namespace = system.UserNamespace()
	}

	k8sclient, err := k8sclient.New(restConfig)
	if err != nil {
		return nil, err
	}

	dialer, err := k8schannel.NewDialer(restConfig, false)
	if err != nil {
		return nil, err
	}

	cfg := rest.CopyConfig(restConfig)
	cfg.APIPath = "/apis"
	cfg.GroupVersion = &apiv1.SchemeGroupVersion
	restconfig.SetScheme(cfg, scheme.Scheme)

	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		return nil, err
	}

	return &IgnoreUninstalled{
		client: &client{
			Namespace:  namespace,
			Client:     k8sclient,
			RESTConfig: restConfig,
			RESTClient: restClient,
			Dialer:     dialer,
		},
	}, nil
}

type AppUpdateOptions struct {
	Annotations      map[string]string
	Labels           map[string]string
	Endpoints        []v1.EndpointBinding
	Volumes          []v1.VolumeBinding
	Secrets          []v1.SecretBinding
	Services         []v1.ServiceBinding
	Ports            []v1.PortBinding
	PublishProtocols []v1.Protocol
	Profiles         []string
	DeployArgs       map[string]interface{}
	DevMode          *bool
	Image            string
}

type LogOptions apiv1.LogOptions

type AppRunOptions struct {
	Name             string
	Annotations      map[string]string
	Labels           map[string]string
	Endpoints        []v1.EndpointBinding
	Volumes          []v1.VolumeBinding
	Secrets          []v1.SecretBinding
	Services         []v1.ServiceBinding
	Ports            []v1.PortBinding
	PublishProtocols []v1.Protocol
	Profiles         []string
	DeployArgs       map[string]interface{}
	DevMode          *bool
}

func (a AppRunOptions) ToUpdate() AppUpdateOptions {
	return AppUpdateOptions{
		Annotations:      a.Annotations,
		Labels:           a.Labels,
		Endpoints:        a.Endpoints,
		Volumes:          a.Volumes,
		Secrets:          a.Secrets,
		Services:         a.Services,
		Ports:            a.Ports,
		PublishProtocols: a.PublishProtocols,
		DeployArgs:       a.DeployArgs,
		DevMode:          a.DevMode,
		Profiles:         a.Profiles,
	}
}

type ImageProgress struct {
	Total    int64  `json:"total,omitempty"`
	Complete int64  `json:"complete,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ImageDetails struct {
	AppImage   v1.AppImage   `json:"appImage,omitempty"`
	AppSpec    *v1.AppSpec   `json:"appSpec,omitempty"`
	Params     *v1.ParamSpec `json:"params,omitempty"`
	ParseError string        `json:"parseError,omitempty"`
}

type Client interface {
	AppList(ctx context.Context) ([]apiv1.App, error)
	AppDelete(ctx context.Context, name string) (*apiv1.App, error)
	AppGet(ctx context.Context, name string) (*apiv1.App, error)
	AppStop(ctx context.Context, name string) error
	AppStart(ctx context.Context, name string) error
	AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error)
	AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error)
	AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error)

	CredentialCreate(ctx context.Context, serverAddress, username, password string) (*apiv1.Credential, error)
	CredentialList(ctx context.Context) ([]apiv1.Credential, error)
	CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error)
	CredentialUpdate(ctx context.Context, serverAddress, username, password string) (*apiv1.Credential, error)
	CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error)

	SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error)
	SecretList(ctx context.Context) ([]apiv1.Secret, error)
	SecretGet(ctx context.Context, name string) (*apiv1.Secret, error)
	SecretExpose(ctx context.Context, name string) (*apiv1.Secret, error)
	SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error)
	SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error)

	ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error)
	ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error)
	ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error)
	ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error)

	VolumeList(ctx context.Context) ([]apiv1.Volume, error)
	VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error)
	VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error)

	ImageList(ctx context.Context) ([]apiv1.Image, error)
	ImageGet(ctx context.Context, name string) (*apiv1.Image, error)
	ImageDelete(ctx context.Context, name string) (*apiv1.Image, error)
	ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (<-chan ImageProgress, error)
	ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (<-chan ImageProgress, error)
	ImageTag(ctx context.Context, image, tag string) error
	ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error)

	BuilderCreate(ctx context.Context) (*apiv1.Builder, error)
	BuilderGet(ctx context.Context) (*apiv1.Builder, error)
	BuilderDelete(ctx context.Context) (*apiv1.Builder, error)
	BuilderDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error)
	BuilderRegistryDialer(ctx context.Context) (func(ctx context.Context) (net.Conn, error), error)

	Info(ctx context.Context) (*apiv1.Info, error)

	GetNamespace() string
	GetClient() kclient.WithWatch
}

type ImagePullOptions struct {
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type ImagePushOptions struct {
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type ImageDetailsOptions struct {
}

type ContainerReplicaExecOptions struct {
	DebugImage string `json:"debugImage,omitempty"`
}

type ContainerReplicaListOptions struct {
	App string `json:"app,omitempty"`
}

type client struct {
	Namespace  string
	Client     kclient.WithWatch
	RESTConfig *rest.Config
	RESTClient *rest.RESTClient
	Dialer     *k8schannel.Dialer
}

func (c *client) GetNamespace() string {
	return c.Namespace
}

func (c *client) GetClient() kclient.WithWatch {
	return c.Client
}
