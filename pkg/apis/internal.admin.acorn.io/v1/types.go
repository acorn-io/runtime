// +k8s:deepcopy-gen=package

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterComputeClassInstance ProjectComputeClassInstance

func (in *ClusterComputeClassInstance) NamespaceScoped() bool {
	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterComputeClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterComputeClassInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectComputeClassInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Description       string              `json:"description,omitempty"`
	CPUScaler         float64             `json:"cpuScaler,omitempty"`
	Default           bool                `json:"default"`
	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	Memory            ComputeClassMemory  `json:"memory,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectComputeClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectComputeClassInstance `json:"items"`
}

type ComputeClassMemory struct {
	Min     string   `json:"min,omitempty"`
	Max     string   `json:"max,omitempty"`
	Default string   `json:"default,omitempty"`
	Values  []string `json:"values,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RegionInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   RegionInstanceSpec   `json:"spec,omitempty"`
	Status RegionInstanceStatus `json:"status,omitempty"`
}

func (in *RegionInstance) ForRegion(region string) bool {
	return in.Spec.RegionName == region
}

func (in *RegionInstance) NamespaceScoped() bool {
	return false
}

type RegionInstanceSpec struct {
	Description string `json:"description,omitempty"`
	AccountName string `json:"accountName,omitempty"`
	Role        string `json:"role,omitempty"`
	RegionName  string `json:"regionName,omitempty"`
}

type RegionInstanceStatus struct {
	ClusterCreated bool `json:"clusterCreated,omitempty"`
	ClusterReady   bool `json:"clusterReady,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type RegionInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RegionInstance `json:"items"`
}
