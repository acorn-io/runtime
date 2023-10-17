package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/acorn-io/baaah/pkg/restconfig"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client/term"
	"github.com/acorn-io/runtime/pkg/k8schannel"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/proxy"
	"github.com/acorn-io/runtime/pkg/scheme"
	"github.com/acorn-io/runtime/pkg/streams"
	"github.com/acorn-io/runtime/pkg/system"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Factory struct {
	client     kclient.WithWatch
	restConfig *rest.Config
	restClient *rest.RESTClient
	dialer     *k8schannel.Dialer
}

func (f *Factory) Namespace(project, namespace string) Client {
	return &DefaultClient{
		Project:    project,
		Namespace:  namespace,
		Client:     f.client,
		RESTConfig: f.restConfig,
		RESTClient: f.restClient,
		Dialer:     f.dialer,
	}
}

func NewClientFactory(restConfig *rest.Config) (*Factory, error) {
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

	return &Factory{
		client:     k8sclient,
		restConfig: restConfig,
		restClient: restClient,
		dialer:     dialer,
	}, nil
}

func New(restConfig *rest.Config, project, namespace string) (Client, error) {
	if namespace == "" {
		namespace = system.DefaultUserNamespace
	}

	f, err := NewClientFactory(restConfig)
	if err != nil {
		return nil, err
	}
	return f.Namespace(project, namespace), nil
}

type AppUpdateOptions struct {
	Annotations              []v1.ScopedLabel
	Labels                   []v1.ScopedLabel
	PublishMode              v1.PublishMode
	Volumes                  []v1.VolumeBinding
	Secrets                  []v1.SecretBinding
	Links                    []v1.ServiceBinding
	Publish                  []v1.PortBinding
	Env                      []v1.NameValue
	Profiles                 []string
	Permissions              []v1.Permissions
	DeployArgs               map[string]any
	Stop                     *bool
	Image                    string
	Replace                  bool // Replace is used to indicate whether the update should be a patch (replace=false: only change specified fields) or a full update (replace=true: reset unspecified fields to defaults)
	AutoUpgrade              *bool
	NotifyUpgrade            *bool
	AutoUpgradeInterval      string
	Memory                   v1.MemoryMap
	ComputeClasses           v1.ComputeClassMap
	Region                   string
	DevSessionClient         *v1.DevSessionInstanceClient
	DevSessionTimeoutSeconds int32
}

type ContainerLogsWriter interface {
	Container(timeStamp metav1.Time, containerName, line string)
}

type LogOptions struct {
	apiv1.LogOptions

	Logger ContainerLogsWriter
}

type AppRunOptions struct {
	Name                string
	Region              string
	Annotations         []v1.ScopedLabel
	Labels              []v1.ScopedLabel
	PublishMode         v1.PublishMode
	Volumes             []v1.VolumeBinding
	Secrets             []v1.SecretBinding
	Links               []v1.ServiceBinding
	Publish             []v1.PortBinding
	Env                 []v1.NameValue
	Profiles            []string
	DeployArgs          map[string]any
	Stop                *bool
	Permissions         []v1.Permissions
	AutoUpgrade         *bool
	NotifyUpgrade       *bool
	AutoUpgradeInterval string
	Memory              v1.MemoryMap
	ComputeClasses      v1.ComputeClassMap
}

func (a AppRunOptions) ToUpdate() AppUpdateOptions {
	return AppUpdateOptions{
		Annotations:         a.Annotations,
		Labels:              a.Labels,
		PublishMode:         a.PublishMode,
		Volumes:             a.Volumes,
		Secrets:             a.Secrets,
		Links:               a.Links,
		Publish:             a.Publish,
		DeployArgs:          a.DeployArgs,
		Stop:                a.Stop,
		Profiles:            a.Profiles,
		Permissions:         a.Permissions,
		Env:                 a.Env,
		AutoUpgrade:         a.AutoUpgrade,
		NotifyUpgrade:       a.NotifyUpgrade,
		AutoUpgradeInterval: a.AutoUpgradeInterval,
		Memory:              a.Memory,
		ComputeClasses:      a.ComputeClasses,
		Region:              a.Region,
	}
}

func (a AppUpdateOptions) ToRun() AppRunOptions {
	return AppRunOptions{
		Annotations:         a.Annotations,
		Labels:              a.Labels,
		PublishMode:         a.PublishMode,
		Volumes:             a.Volumes,
		Secrets:             a.Secrets,
		Links:               a.Links,
		Publish:             a.Publish,
		DeployArgs:          a.DeployArgs,
		Profiles:            a.Profiles,
		Permissions:         a.Permissions,
		Env:                 a.Env,
		AutoUpgrade:         a.AutoUpgrade,
		NotifyUpgrade:       a.NotifyUpgrade,
		AutoUpgradeInterval: a.AutoUpgradeInterval,
		Memory:              a.Memory,
		ComputeClasses:      a.ComputeClasses,
	}
}

type ImageProgress struct {
	Total       int64  `json:"total,omitempty"`
	Complete    int64  `json:"complete,omitempty"`
	Error       string `json:"error,omitempty"`
	CurrentTask string `json:"currentTask,omitempty"`
}

type ImageDetails struct {
	AppImage        v1.AppImage         `json:"appImage,omitempty"`
	AppSpec         *v1.AppSpec         `json:"appSpec,omitempty"`
	Params          *v1.ParamSpec       `json:"params,omitempty"`
	ImageName       string              `json:"imageName,omitempty"`
	SignatureDigest string              `json:"signatureDigest,omitempty"`
	Readme          string              `json:"readme,omitempty"`
	ParseError      string              `json:"parseError,omitempty"`
	Permissions     []v1.Permissions    `json:"permissions,omitempty"`
	NestedImages    []apiv1.NestedImage `json:"nestedImages,omitempty"`
}

type PortForwardDialer func(ctx context.Context) (net.Conn, error)

type Client interface {
	AppList(ctx context.Context) ([]apiv1.App, error)
	AppDelete(ctx context.Context, name string) (*apiv1.App, error)
	AppGet(ctx context.Context, name string) (*apiv1.App, error)
	AppStop(ctx context.Context, name string) error
	AppStart(ctx context.Context, name string) error
	AppRun(ctx context.Context, image string, opts *AppRunOptions) (*apiv1.App, error)
	AppUpdate(ctx context.Context, name string, opts *AppUpdateOptions) (*apiv1.App, error)
	AppLog(ctx context.Context, name string, opts *LogOptions) (<-chan apiv1.LogMessage, error)
	AppInfo(ctx context.Context, name string) (string, error)
	AppConfirmUpgrade(ctx context.Context, name string) error
	AppPullImage(ctx context.Context, name string) error
	AppIgnoreDeleteCleanup(ctx context.Context, name string) error

	DevSessionRenew(ctx context.Context, name string, client v1.DevSessionInstanceClient) error
	DevSessionRelease(ctx context.Context, name string) error

	CredentialCreate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error)
	CredentialList(ctx context.Context) ([]apiv1.Credential, error)
	CredentialGet(ctx context.Context, serverAddress string) (*apiv1.Credential, error)
	CredentialUpdate(ctx context.Context, serverAddress, username, password string, skipChecks bool) (*apiv1.Credential, error)
	CredentialDelete(ctx context.Context, serverAddress string) (*apiv1.Credential, error)

	SecretCreate(ctx context.Context, name, secretType string, data map[string][]byte) (*apiv1.Secret, error)
	SecretList(ctx context.Context) ([]apiv1.Secret, error)
	SecretGet(ctx context.Context, name string) (*apiv1.Secret, error)
	SecretReveal(ctx context.Context, name string) (*apiv1.Secret, error)
	SecretUpdate(ctx context.Context, name string, data map[string][]byte) (*apiv1.Secret, error)
	SecretDelete(ctx context.Context, name string) (*apiv1.Secret, error)

	ContainerReplicaList(ctx context.Context, opts *ContainerReplicaListOptions) ([]apiv1.ContainerReplica, error)
	ContainerReplicaGet(ctx context.Context, name string) (*apiv1.ContainerReplica, error)
	ContainerReplicaDelete(ctx context.Context, name string) (*apiv1.ContainerReplica, error)
	ContainerReplicaExec(ctx context.Context, name string, args []string, tty bool, opts *ContainerReplicaExecOptions) (*term.ExecIO, error)
	ContainerReplicaPortForward(ctx context.Context, name string, port int) (PortForwardDialer, error)

	JobList(ctx context.Context, opts *JobListOptions) ([]apiv1.Job, error)
	JobGet(ctx context.Context, name string) (*apiv1.Job, error)
	JobRestart(ctx context.Context, name string) error

	VolumeList(ctx context.Context) ([]apiv1.Volume, error)
	VolumeGet(ctx context.Context, name string) (*apiv1.Volume, error)
	VolumeDelete(ctx context.Context, name string) (*apiv1.Volume, error)

	ImageList(ctx context.Context) ([]apiv1.Image, error)
	ImageGet(ctx context.Context, name string) (*apiv1.Image, error)
	ImageDelete(ctx context.Context, name string, opts *ImageDeleteOptions) (*apiv1.Image, []string, error) // returns the modified/deleted image and a list of deleted tags
	ImagePush(ctx context.Context, tagName string, opts *ImagePushOptions) (<-chan ImageProgress, error)
	ImagePull(ctx context.Context, name string, opts *ImagePullOptions) (<-chan ImageProgress, error)
	ImageTag(ctx context.Context, image, tag string) error
	ImageDetails(ctx context.Context, imageName string, opts *ImageDetailsOptions) (*ImageDetails, error)

	ImageSign(ctx context.Context, image string, payload []byte, signatureB64 string, opts *ImageSignOptions) (*apiv1.ImageSignature, error)
	ImageVerify(ctx context.Context, image string, opts *ImageVerifyOptions) (*apiv1.ImageSignature, error)

	AcornImageBuildGet(ctx context.Context, name string) (*apiv1.AcornImageBuild, error)
	AcornImageBuildList(ctx context.Context) ([]apiv1.AcornImageBuild, error)
	AcornImageBuildDelete(ctx context.Context, name string) (*apiv1.AcornImageBuild, error)
	AcornImageBuild(ctx context.Context, file string, opts *AcornImageBuildOptions) (*v1.AppImage, error)

	ProjectGet(ctx context.Context, name string) (*apiv1.Project, error)
	ProjectList(ctx context.Context) ([]apiv1.Project, error)
	ProjectCreate(ctx context.Context, name, defaultRegion string, supportedRegions []string) (*apiv1.Project, error)
	ProjectUpdate(ctx context.Context, project *apiv1.Project, defaultRegion string, supportedRegions []string) (*apiv1.Project, error)
	ProjectDelete(ctx context.Context, name string) (*apiv1.Project, error)

	VolumeClassList(ctx context.Context) ([]apiv1.VolumeClass, error)
	VolumeClassGet(ctx context.Context, name string) (*apiv1.VolumeClass, error)

	Info(ctx context.Context) ([]apiv1.Info, error)

	ComputeClassList(ctx context.Context) ([]apiv1.ComputeClass, error)
	ComputeClassGet(ctx context.Context, name string) (*apiv1.ComputeClass, error)

	RegionList(ctx context.Context) ([]apiv1.Region, error)
	RegionGet(ctx context.Context, name string) (*apiv1.Region, error)

	EventStream(ctx context.Context, opts *EventStreamOptions) (<-chan apiv1.Event, error)

	GetProject() string
	GetNamespace() string
	GetClient() (kclient.WithWatch, error)

	KubeProxyAddress(ctx context.Context, opts *KubeProxyAddressOptions) (string, error)
	KubeConfig(ctx context.Context, opts *KubeProxyAddressOptions) ([]byte, error)
}

type CredentialLookup func(ctx context.Context, serverAddress string) (*apiv1.RegistryAuth, bool, error)

type AcornImageBuildOptions struct {
	BuilderName string
	Credentials CredentialLookup
	Cwd         string
	Platforms   []v1.Platform
	Args        map[string]any
	Profiles    []string
	Streams     *streams.Output
}

func (a *AcornImageBuildOptions) complete() (_ *AcornImageBuildOptions, err error) {
	var newOpt AcornImageBuildOptions
	if a != nil {
		newOpt = *a
	}
	if newOpt.Cwd == "" {
		newOpt.Cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}
	if newOpt.Streams == nil {
		newOpt.Streams = streams.CurrentOutput()
	}
	return &newOpt, nil
}

type ImagePullOptions struct {
	Auth *apiv1.RegistryAuth `json:"auth,omitempty"`
}

type ImagePushOptions struct {
	Auth *apiv1.RegistryAuth `json:"auth,omitempty"`
}

type ImageDetailsOptions struct {
	NestedDigest  string
	Profiles      []string
	DeployArgs    map[string]any
	Auth          *apiv1.RegistryAuth
	IncludeNested bool
	// NoDefaultRegistry - if true, indicates that no default container registry should be assumed when getting image details
	NoDefaultRegistry bool
}

type ImageDeleteOptions struct {
	Force bool `json:"force,omitempty"`
}

type ContainerReplicaExecOptions struct {
	DebugImage string `json:"debugImage,omitempty"`
}

type ContainerReplicaListOptions struct {
	App string `json:"app,omitempty"`
}

type KubeProxyAddressOptions struct {
	Region string `json:"region,omitempty"`
}

type JobListOptions struct {
	App string `json:"app,omitempty"`
}

type EventStreamOptions struct {
	Tail            int    `json:"tail,omitempty"`
	Follow          bool   `json:"follow,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	Since           string `json:"since,omitempty"`
	Until           string `json:"until,omitempty"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

type ImageSignOptions struct {
	PublicKey string              `json:"publicKeys,omitempty"`
	Auth      *apiv1.RegistryAuth `json:"auth,omitempty"`
}

type ImageVerifyOptions struct {
	PublicKey    string              `json:"publicKeys,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
	Auth         *apiv1.RegistryAuth `json:"auth,omitempty"`
	NoVerifyName bool                `json:"noVerifyName,omitempty"`
}

func (o EventStreamOptions) ListOptions() *kclient.ListOptions {
	fieldSet := make(fields.Set)
	if o.Prefix != "" {
		fieldSet["prefix"] = o.Prefix
	}
	if o.Since != "" {
		fieldSet["since"] = o.Since
	}
	if o.Until != "" {
		fieldSet["until"] = o.Until
	}

	// Set details selector to get details from older runtime APIs that don't return details by default.
	fieldSet["details"] = strconv.FormatBool(true)

	return &kclient.ListOptions{
		Limit:         int64(o.Tail),
		FieldSelector: fieldSet.AsSelector(),
		Raw: &metav1.ListOptions{
			ResourceVersion: o.ResourceVersion,
		},
	}
}

type DefaultClient struct {
	Project    string
	Namespace  string
	Client     kclient.WithWatch
	RESTConfig *rest.Config
	RESTClient *rest.RESTClient
	Dialer     *k8schannel.Dialer
}

func generateKubeConfig(restConfig *rest.Config) ([]byte, error) {
	config := &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{
			"acorn": {
				Server:                   restConfig.Host,
				CertificateAuthorityData: restConfig.TLSClientConfig.CAData,
			},
		},
		AuthInfos: map[string]*clientcmdapi.AuthInfo{
			"auth": {
				Token: restConfig.BearerToken,
			},
		},
		Contexts: map[string]*clientcmdapi.Context{
			"default": {
				Cluster:  "acorn",
				AuthInfo: "auth",
			},
		},
		CurrentContext: "default",
	}

	return clientcmd.Write(*config)
}

func (c *DefaultClient) getRESTConfig(ctx context.Context, opts *KubeProxyAddressOptions) (*rest.Config, error) {
	if opts == nil || opts.Region == "" {
		return c.RESTConfig, nil
	}

	client, err := rest.HTTPClientFor(c.RESTConfig)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(map[string]any{
		"kind":       "AccessConfig",
		"apiVersion": "aws.account.manager.acorn.io/v1",
		"regionName": opts.Region,
	})
	if err != nil {
		return nil, err
	}
	// Could this be prettier
	resp, err := client.Post(c.RESTConfig.Host+"/apis/aws.account.manager.acorn.io/v1/accessconfigs",
		"application/json", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("invalid response looking up kubeconfig %d: %s", resp.StatusCode, body)
	}

	parsed := struct {
		Config []byte
	}{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}

	return clientcmd.RESTConfigFromKubeConfig(parsed.Config)
}

func (c *DefaultClient) KubeConfig(ctx context.Context, opts *KubeProxyAddressOptions) ([]byte, error) {
	restConfig, err := c.getRESTConfig(ctx, opts)
	if err != nil {
		return nil, err
	}
	return generateKubeConfig(restConfig)
}

func (c *DefaultClient) KubeProxyAddress(ctx context.Context, opts *KubeProxyAddressOptions) (string, error) {
	restConfig, err := c.getRESTConfig(ctx, opts)
	if err != nil {
		return "", err
	}

	handler, err := proxy.Handler(restConfig)
	if err != nil {
		return "", err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		return "", err
	}

	srv := &http.Server{
		Handler: handler,
		BaseContext: func(listener net.Listener) context.Context {
			return ctx
		},
	}
	go func() {
		_ = srv.Serve(ln)
		os.Exit(1)
	}()
	return "http://" + ln.Addr().String(), nil
}

func (c *DefaultClient) GetProject() string {
	return c.Project
}

func (c *DefaultClient) GetNamespace() string {
	return c.Namespace
}

func (c *DefaultClient) GetClient() (kclient.WithWatch, error) {
	return c.Client, nil
}
