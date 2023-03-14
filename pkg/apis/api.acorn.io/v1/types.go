package v1

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type App struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   v1.AppInstanceSpec   `json:"spec,omitempty"`
	Status v1.AppInstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []App `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerReplica struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ContainerReplicaSpec   `json:"spec,omitempty"`
	Status ContainerReplicaStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerReplicaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ContainerReplica `json:"items"`
}

type ContainerReplicaSpec struct {
	AppName       string `json:"appName,omitempty"`
	JobName       string `json:"jobName,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	SidecarName   string `json:"sidecarName,omitempty"`

	Dirs        map[string]v1.VolumeMount `json:"dirs,omitempty"`
	Files       map[string]v1.File        `json:"files,omitempty"`
	Image       string                    `json:"image,omitempty"`
	Build       *v1.Build                 `json:"build,omitempty"`
	Command     []string                  `json:"command,omitempty"`
	Interactive bool                      `json:"interactive,omitempty"`
	Entrypoint  []string                  `json:"entrypoint,omitempty"`
	Environment []v1.EnvVar               `json:"environment,omitempty"`
	WorkingDir  string                    `json:"workingDir,omitempty"`
	Ports       []v1.PortDef              `json:"ports,omitempty"`

	// Init is only available on sidecars
	Init bool `json:"init,omitempty"`

	// Sidecars are not available on sidecars
	Sidecars map[string]v1.Container `json:"sidecars,omitempty"`
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

	Region string `json:"region,omitempty"`
}

// EnsureRegion checks or sets the region of a ContainerReplica.
// If a ContainerReplica's region is unset, EnsureRegion sets it to the given region and returns true.
// Otherwise, it returns true if and only if the ContainerReplica belongs to the given region.
func (in *ContainerReplica) EnsureRegion(region string) bool {
	// If the region of a Container Replica is not set, then it hasn't been synced yet. In this case, we assume that it is in
	// the same region as the app, and return true.
	if in.Status.Region == "" {
		in.Status.Region = region
	}

	return in.Status.Region == region
}

func (in *ContainerReplica) GetRegion() string {
	return in.Status.Region
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Project    string   `json:"project,omitempty"`
	Repository string   `json:"repo,omitempty"`
	Digest     string   `json:"digest,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePush struct {
	metav1.TypeMeta `json:",inline"`
	Auth            *RegistryAuth `json:"auth,omitempty"`
}

type RegistryAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePull struct {
	metav1.TypeMeta `json:",inline"`
	Auth            *RegistryAuth `json:"auth,omitempty"`
}

type LogMessage struct {
	Line          string      `json:"line,omitempty"`
	AppName       string      `json:"appName,omitempty"`
	ContainerName string      `json:"containerName,omitempty"`
	Time          metav1.Time `json:"time,omitempty"`
	Error         string      `json:"error,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type LogOptions struct {
	metav1.TypeMeta `json:",inline"`

	Tail             *int64 `json:"tailLines,omitempty"`
	Follow           bool   `json:"follow,omitempty"`
	ContainerReplica string `json:"containerReplica,omitempty"`
	Since            string `json:"since,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppPullImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ConfirmUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageDetails struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Input Params
	DeployArgs v1.GenericMap `json:"deployArgs,omitempty"`
	Profiles   []string      `json:"profiles,omitempty"`

	// Output Params
	AppImage   v1.AppImage   `json:"appImage,omitempty"`
	AppSpec    *v1.AppSpec   `json:"appSpec,omitempty"`
	Params     *v1.ParamSpec `json:"params,omitempty"`
	ParseError string        `json:"parseError,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageTag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Tag string `json:"tag,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Image `json:"items"`
}

type VolumeCreateOptions struct {
	AccessModes []v1.AccessMode `json:"accessModes,omitempty"`
	Class       string          `json:"class,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Volume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   VolumeSpec   `json:"spec,omitempty"`
	Status VolumeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Volume `json:"items"`
}

type VolumeSpec struct {
	Capacity    *resource.Quantity `json:"capacity,omitempty"`
	AccessModes []v1.AccessMode    `json:"accessModes,omitempty"`
	Class       string             `json:"class,omitempty"`
}

type VolumeStatus struct {
	AppName      string        `json:"appName,omitempty"`
	AppNamespace string        `json:"appNamespace,omitempty"`
	VolumeName   string        `json:"volumeName,omitempty"`
	Status       string        `json:"status,omitempty"`
	Columns      VolumeColumns `json:"columns,omitempty"`
	Region       string        `json:"region,omitempty"`
}

type VolumeColumns struct {
	AccessModes string `json:"accessModes,omitempty"`
}

// EnsureRegion checks or sets the region of a ContainerReplica.
// If a ContainerReplica's region is unset, EnsureRegion sets it to the given region and returns true.
// Otherwise, it returns true if and only if the ContainerReplica belongs to the given region.
func (in *Volume) EnsureRegion(region string) bool {
	// If the region of a volume is not set, then it hasn't been synced yet. In this case, we assume that the volume is in
	// the same region as the app, and return true.
	if in.Status.Region == "" {
		in.Status.Region = region
	}

	return in.Status.Region == region
}

// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerReplicaExecOptions struct {
	metav1.TypeMeta `json:",inline"`

	Command    []string `json:"command,omitempty"`
	TTY        bool     `json:"tty,omitempty"`
	DebugImage string   `json:"debugImage,omitempty"`
}

const (
	SecretTypeCredential = "acorn.io/credential"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Credential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	ServerAddress string  `json:"serverAddress,omitempty"`
	Username      string  `json:"username,omitempty"`
	Password      *string `json:"password,omitempty"`
	SkipChecks    bool    `json:"skipChecks,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CredentialList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Credential `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Secret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Type string            `json:"type,omitempty"`
	Data map[string][]byte `json:"data,omitempty"`
	Keys []string          `json:"keys,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type SecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Secret `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Info struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec InfoSpec `json:"spec,omitempty"`
}

// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderPortOptions struct {
	metav1.TypeMeta `json:",inline"`
}

type InfoSpec struct {
	Version                string            `json:"version"`
	Tag                    string            `json:"tag"`
	GitCommit              string            `json:"gitCommit"`
	Dirty                  bool              `json:"dirty"`
	ControllerImage        string            `json:"controllerImage"`
	APIServerImage         string            `json:"apiServerImage,omitempty"`
	PublicKeys             []EncryptionKey   `json:"publicKeys,omitempty"`
	Config                 Config            `json:"config"`
	UserConfig             Config            `json:"userConfig"`
	LetsEncryptCertificate string            `json:"letsEncryptCertificate,omitempty"`
	ExtraData              map[string]string `json:"extraData,omitempty"`
}

type Config struct {
	// Do not set omitEmpty on the json fields.  Also make strings and bool a
	// pointer unless the default value (false, "") is not a valid configuration

	// For repeatable flags, ensure the struct and json fields are plural and the flag name is singular.
	// See ClusterDomains as an example.

	IngressClassName               *string  `json:"ingressClassName" usage:"The ingress class name to assign to all created ingress resources (default '')"`
	ClusterDomains                 []string `json:"clusterDomains" name:"cluster-domain" usage:"The externally addressable cluster domain (default .on-acorn.io)"`
	LetsEncrypt                    *string  `json:"letsEncrypt" name:"lets-encrypt" usage:"enabled|disabled|staging. If enabled, acorn generated endpoints will be secured using TLS certificate from Let's Encrypt. Staging uses Let's Encrypt's staging environment. (default disabled)"`
	LetsEncryptEmail               string   `json:"letsEncryptEmail" name:"lets-encrypt-email" usage:"Required if --lets-encrypt=enabled. The email address to use for Let's Encrypt registration(default '')"`
	LetsEncryptTOSAgree            *bool    `json:"letsEncryptTOSAgree" name:"lets-encrypt-tos-agree" usage:"Required if --lets-encrypt=enabled. If true, you agree to the Let's Encrypt terms of service (default false)"`
	SetPodSecurityEnforceProfile   *bool    `json:"setPodSecurityEnforceProfile" usage:"Set the PodSecurity profile on created namespaces (default true)"`
	PodSecurityEnforceProfile      string   `json:"podSecurityEnforceProfile" usage:"The name of the PodSecurity profile to set (default baseline)" wrangler:"nullable"`
	HttpEndpointPattern            *string  `json:"httpEndpointPattern" name:"http-endpoint-pattern" usage:"Go template for formatting application http endpoints. Valid variables to use are: App, Container, Namespace, Hash and ClusterDomain. (default pattern is {{hashConcat 8 .Container .App .Namespace | truncate}}.{{.ClusterDomain}})" wrangler:"nullable"`
	InternalClusterDomain          string   `json:"internalClusterDomain" usage:"The Kubernetes internal cluster domain (default svc.cluster.local)" wrangler:"nullable"`
	AcornDNS                       *string  `json:"acornDNS" name:"acorn-dns" usage:"enabled|disabled|auto. If enabled, containers created by Acorn will get public FQDNs. Auto functions as disabled if a custom clusterDomain has been supplied (default auto)"`
	AcornDNSEndpoint               *string  `json:"acornDNSEndpoint" name:"acorn-dns-endpoint" usage:"The URL to access the Acorn DNS service"`
	AutoUpgradeInterval            *string  `json:"autoUpgradeInterval" name:"auto-upgrade-interval" usage:"For apps configured with automatic upgrades enabled, the interval at which to check for new versions. Upgrade intervals configured at the application level cannot be smaller than this. (default '5m' - 5 minutes)"`
	RecordBuilds                   *bool    `json:"recordBuilds" name:"record-builds" usage:"Keep a record of each acorn build that happens"`
	PublishBuilders                *bool    `json:"publishBuilders" name:"publish-builders" usage:"Publish the builders through ingress to so build traffic does not traverse the api-server"`
	BuilderPerProject              *bool    `json:"builderPerProject" name:"builder-per-project" usage:"Create a dedicated builder per project"`
	InternalRegistryPrefix         *string  `json:"internalRegistryPrefix" name:"internal-registry-prefix" usage:"The image prefix to use when pushing internal images (example ghcr.io/my-org/)"`
	IgnoreUserLabelsAndAnnotations *bool    `json:"ignoreUserLabelsAndAnnotations" name:"ignore-user-labels-and-annotations" usage:"Don't propagate user-defined labels and annotations to dependent objects"`
	AllowUserLabels                []string `json:"allowUserLabels" name:"allow-user-label" usage:"Allow these labels to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true"`
	AllowUserAnnotations           []string `json:"allowUserAnnotations" name:"allow-user-annotation" usage:"Allow these annotations to propagate to dependent objects, no effect if --ignore-user-labels-and-annotations not true"`
	WorkloadMemoryDefault          *int64   `json:"workloadMemoryDefault" name:"workload-memory-default" quantity:"true" usage:"Set the default memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and \".\" and \"_\" seperators (default 0)" short:"m"`
	WorkloadMemoryMaximum          *int64   `json:"workloadMemoryMaximum" name:"workload-memory-maximum" quantity:"true" usage:"Set the maximum memory for acorn workloads. Accepts binary suffixes (Ki, Mi, Gi, etc) and \".\" and \"_\" seperators (default 0)"`
	UseCustomCABundle              *bool    `json:"useCustomCABundle" name:"use-custom-ca-bundle" usage:"Use CA bundle for admin supplied secret for all acorn control plane components. Defaults to false."`
	PropagateProjectAnnotations    []string `json:"propagateProjectAnnotations" name:"propagate-project-annotation" usage:"The list of keys of annotations to propagate from acorn project to app namespaces"`
	PropagateProjectLabels         []string `json:"propagateProjectLabels" name:"propagate-project-label" usage:"The list of keys of labels to propagate from acorn project to app namespaces"`
	ManageVolumeClasses            *bool    `json:"manageVolumeClasses" name:"manage-volume-classes" usage:"Manually manage volume classes rather than sync with storage classes, setting to 'true' will delete Acorn-created volume classes"`
	NetworkPolicies                *bool    `json:"networkPolicies" name:"network-policies" usage:"Create Kubernetes NetworkPolicies which block cross-project network traffic (default true)"`
	IngressControllerNamespace     *string  `json:"ingressControllerNamespace" name:"ingress-controller-namespace" usage:"The namespace where the ingress controller runs - used to secure published HTTP ports with NetworkPolicies."`
	AllowTrafficFromNamespace      []string `json:"allowTrafficFromNamespace" name:"allow-traffic-from-namespace" usage:"Namespaces that are allowed to send network traffic to all Acorn apps"`
	ServiceLBAnnotations           []string `json:"serviceLBAnnotations" name:"service-lb-annotation" usage:"Annotation to add to the service of type LoadBalancer. Defaults to empty. (example key=value)"`
}

type EncryptionKey struct {
	KeyID       string            `json:"keyID"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type InfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Info `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ProjectSpec   `json:"spec,omitempty"`
	Status            ProjectStatus `json:"status,omitempty"`
}

type ProjectSpec struct {
	DefaultRegion    string   `json:"defaultRegion,omitempty"`
	SupportedRegions []string `json:"supportedRegions,omitempty"`
}

type ProjectStatus struct {
	Namespace     string `json:"namespace,omitempty"`
	DefaultRegion string `json:"defaultRegion,omitempty"`
}

func (in *Project) NamespaceScoped() bool {
	return false
}

func (in *Project) HasRegion(region string) bool {
	return region == "" || in.Status.DefaultRegion == region || slices.Contains(in.Spec.SupportedRegions, region)
}

func (in *Project) GetRegion() string {
	if in.Spec.DefaultRegion != "" {
		return in.Spec.DefaultRegion
	}
	return in.Status.DefaultRegion
}

func (in *Project) SetDefaultRegion(region string) {
	if in.Spec.DefaultRegion == "" {
		in.Status.DefaultRegion = region
	} else {
		in.Status.DefaultRegion = ""
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Builder v1.BuilderInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Builder `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornImageBuild v1.AcornImageBuildInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornImageBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AcornImageBuild `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VolumeClass adminv1.ProjectVolumeClassInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type VolumeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VolumeClass `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Service v1.ServiceInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Service `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRule v1.ImageAllowRuleInstance

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageAllowRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageAllowRule `json:"items"`
}
