package v1

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

type ClusterSpec struct {
	Address string
}

type ClusterStatus struct {
	Available bool            `json:"available"`
	Installed bool            `json:"installed"`
	Error     string          `json:"error,omitempty"`
	Provider  string          `json:"provider"`
	Info      *apiv1.InfoSpec `json:"info,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

const (
	InstallModeConfig    = "config"
	InstallModeResources = "resources"
	InstallModeBoth      = "both"
)

func (m InstallMode) DoConfig() bool {
	return m == InstallModeBoth || m == InstallModeConfig
}

func (m InstallMode) DoResources() bool {
	return m == InstallModeBoth || m == InstallModeResources
}

type InstallMode string

type Install struct {
	Config apiv1.Config `json:"config,omitempty"`
	Image  string       `json:"image,omitempty"`
	Mode   InstallMode  `wrangler:"type=enum"`
}
