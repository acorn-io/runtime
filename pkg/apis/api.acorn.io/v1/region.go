package v1

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RegionConditionClusterReady = "ClusterReady"

	LocalRegion = "local"
	AllRegions  = "*"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Region struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RegionSpec   `json:"spec,omitempty"`
	Status RegionStatus `json:"status,omitempty"`
}

type RegionSpec struct {
	Description string `json:"description,omitempty"`
	RegionName  string `json:"regionName,omitempty"`
}

type RegionStatus struct {
	Conditions []v1.Condition `json:"conditions,omitempty"`
}

func (in *Region) NamespaceScoped() bool {
	return false
}

func (in *Region) HasRegion(region string) bool {
	return in.Spec.RegionName == region
}

func (in *Region) Conditions() *[]v1.Condition {
	return &in.Status.Conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RegionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Region `json:"items"`
}
