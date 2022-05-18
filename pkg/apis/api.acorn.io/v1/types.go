package v1

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
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
	Ports       []v1.Port                 `json:"ports,omitempty"`

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

	Digest string   `json:"digest,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePush struct {
	metav1.TypeMeta `json:",inline"`

	PullSecrets []string `json:"pullSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImagePull struct {
	metav1.TypeMeta `json:",inline"`

	PullSecrets []string `json:"pullSecrets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageDetails struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	PullSecrets []string    `json:"pullSecrets,omitempty"`
	AppImage    v1.AppImage `json:"appImage,omitempty"`
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
