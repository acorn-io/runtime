package client

import (
	"context"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
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

type AppRunOptions struct {
	Name             string
	Annotations      map[string]string
	Labels           map[string]string
	Endpoints        []v1.EndpointBinding
	Volumes          []v1.VolumeBinding
	Secrets          []v1.SecretBinding
	DeployParams     map[string]interface{}
	ImagePullSecrets []string
}

type ImageProgress struct {
	Total    int64  `json:"total,omitempty"`
	Complete int64  `json:"complete,omitempty"`
	Error    string `json:"error,omitempty"`
}

type ImageDetails struct {
	AppImage v1.AppImage `json:"appImage,omitempty"`
}

type Client interface {
	AppList(ctx context.Context) ([]apiv1.App, error)
	AppDelete(ctx context.Context, name string) (*apiv1.App, error)
	AppGet(ctx context.Context, name string) (*apiv1.App, error)
	AppStop(ctx context.Context, name string) error
	AppStart(ctx context.Context, name string) error
	AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error)

	CredentialCreate(ctx context.Context, serverAddress, username, password string) (*apiv1.Credential, error)
	CredentialList(ctx context.Context) ([]apiv1.Credential, error)
	CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error)
	CredentialUpdate(ctx context.Context, serverAddress, username, password string) (*apiv1.Credential, error)
	CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error)

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
}

type ImagePullOptions struct {
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type ImagePushOptions struct {
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type ImageDetailsOptions struct {
	PullSecrets []string `json:"pullSecrets,omitempty"`
}

type ContainerReplicaExecOptions struct {
	DebugImage string `json:"debugImage,omitempty"`
}

type ContainerReplicaListOptions struct {
	App string `json:"app,omitempty"`
}

type client struct {
	Namespace  string `json:"namespace,omitempty"`
	Client     kclient.WithWatch
	RESTConfig *rest.Config
	RESTClient *rest.RESTClient
	Dialer     *k8schannel.Dialer
}
