package v1

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Digest     string `json:"digest,omitempty"`
	Repository string `json:"repository,omitempty"`
	Tag        string `json:"tag,omitempty"`
	Reference  string `json:"reference,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePush struct {
	metav1.TypeMeta `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePull struct {
	metav1.TypeMeta `json:",inline"`
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

	TailLines *int64 `json:"tailLines,omitempty"`
	Follow    bool   `json:"follow,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageDetails struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	DeployArgs v1.GenericMap `json:"deployArgs,omitempty"`
	Profiles   []string      `json:"profiles,omitempty"`

	AppImage   v1.AppImage   `json:"appImage,omitempty"`
	AppSpec    *v1.AppSpec   `json:"appSpec,omitempty"`
	Params     *v1.ParamSpec `json:"params,omitempty"`
	ParseError string        `json:"parseError,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageTag struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	TagName string `json:"tagName,omitempty"`
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
}

type VolumeColumns struct {
	AccessModes string `json:"accessModes,omitempty"`
}

// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ContainerReplicaExecOptions struct {
	metav1.TypeMeta `json:",inline"`

	Command    []string `json:"command,omitempty"`
	TTY        bool     `json:"tty,omitempty"`
	DebugImage string   `json:"debugImage,omitempty"`
}

type CredentialStorageType string

const (
	CredentialStorageTypeCluster = CredentialStorageType("cluster")
	CredentialStorageTypeClient  = CredentialStorageType("client")
	SecretTypeCredential         = "acorn.io/credential"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Credential struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	ServerAddress string                `json:"serverAddress,omitempty"`
	Storage       CredentialStorageType `json:"storage,omitempty"`
	Username      string                `json:"username,omitempty"`
	Password      string                `json:"password,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Builder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Status BuilderStatus `json:"status,omitempty"`
}

type BuilderStatus struct {
	Ready bool `json:"ready,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Builder `json:"items"`
}

// +k8s:conversion-gen:explicit-from=net/url.Values
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderPortOptions struct {
	metav1.TypeMeta `json:",inline"`
}

type InfoSpec struct {
	Version         string              `json:"version"`
	Tag             string              `json:"tag"`
	GitCommit       string              `json:"gitCommit"`
	Dirty           bool                `json:"dirty"`
	ControllerImage string              `json:"controllerImage"`
	APIServerImage  string              `json:"apiServerImage,omitempty"`
	PublicKeys      []EncryptionKeySpec `json:"publicKeys,omitempty"`
	Config          Config              `json:"config"`
	UserConfig      Config              `json:"userConfig"`
}

type Config struct {
	// Do not set omitEmpty on the json fields.  Also make strings and bool a
	// pointer unless the default value (false, "") is not a valid configuration

	IngressClassName             *string        `json:"ingressClassName" usage:"The ingress class name to assign to all created ingress resources (default '')"`
	ClusterDomains               []string       `json:"clusterDomains" name:"cluster-domain" usage:"The externally addressable cluster domain (default .on-acorn.io)"`
	TLSEnabled                   *bool          `json:"tlsEnabled" name:"tls-enabled" usage:"If true HTTPS URLs will be rendered for HTTP endpoint URLs (default false)"`
	SetPodSecurityEnforceProfile *bool          `json:"setPodSecurityEnforceProfile" usage:"Set the PodSecurity profile on created namespaces (default true)"`
	PodSecurityEnforceProfile    string         `json:"podSecurityEnforceProfile" usage:"The name of the PodSecurity profile to set (default baseline)" wrangler:"nullable"`
	DefaultPublishMode           v1.PublishMode `json:"defaultPublishMode" usage:"If no publish mode is set default to this value (default user)" wrangler:"nullable,options=all|none|defined"`
	InternalClusterDomain        string         `json:"internalClusterDomain" usage:"The Kubernetes internal cluster domain (default svc.cluster.local)" wrangler:"nullable"`
	AcornDNS                     *string        `json:"acornDNS" name:"acorn-dns" usage:"enabled|disabled|auto. If enabled, containers created by Acorn will get public FQDNs. Auto functions as disabled if a custom clusterDomain has been supplied (default auto)"`
	AcornDNSEndpoint             *string        `json:"acornDNSEndpoint" name:"acorn-dns-endpoint" usage:"The URL to access the Acorn DNS service"`
}

type EncryptionKeySpec struct {
	KeyID       string            `json:"keyID"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type InfoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Info `json:"items"`
}
