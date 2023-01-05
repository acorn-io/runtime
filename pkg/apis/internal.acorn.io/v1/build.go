package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	AcornImageBuildInstanceConditionBuild = "build"
)

type ContainerImageBuilderSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
	// Sidecars is only populated for non-sidecar containers
	Sidecars map[string]ContainerImageBuilderSpec `json:"sidecars,omitempty"`
}

type ImageBuilderSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type BuilderSpec struct {
	Platforms  []Platform                           `json:"platforms,omitempty"`
	Containers map[string]ContainerImageBuilderSpec `json:"containers,omitempty"`
	Jobs       map[string]ContainerImageBuilderSpec `json:"jobs,omitempty"`
	Images     map[string]ImageBuilderSpec          `json:"images,omitempty"`
}

type ParamSpec struct {
	Params   []Param   `json:"params,omitempty"`
	Profiles []Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type Param struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty" wrangler:"options=string|int|float|bool|object|array"`
	Schema      string `json:"schema,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornImageBuildInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   AcornImageBuildInstanceSpec   `json:"spec,omitempty"`
	Status AcornImageBuildInstanceStatus `json:"status,omitempty"`
}

type AcornImageBuildInstanceSpec struct {
	BuilderName string     `json:"builderName,omitempty" wrangler:"required"`
	Acornfile   string     `json:"acornfile,omitempty"`
	Platforms   []Platform `json:"platforms,omitempty"`
	Args        GenericMap `json:"args,omitempty"`
	Profiles    []string   `json:"profiles,omitempty"`
	VCS         VCS        `json:"vcs,omitempty"`
}

type AcornImageBuildInstanceStatus struct {
	ObservedGeneration int64       `json:"observedGeneration,omitempty"`
	BuildURL           string      `json:"buildURL,omitempty"`
	Token              string      `json:"token,omitempty"`
	AppImage           AppImage    `json:"appImage,omitempty"`
	Conditions         []Condition `json:"conditions,omitempty"`
	BuildError         string      `json:"buildError,omitempty"`
}

func (in *AcornImageBuildInstance) Conditions() *[]Condition {
	return &in.Status.Conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AcornImageBuildInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AcornImageBuildInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Status BuilderInstanceStatus `json:"status,omitempty"`
}

type BuilderInstanceStatus struct {
	Ready       bool   `json:"ready,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type BuilderInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BuilderInstance `json:"items"`
}
