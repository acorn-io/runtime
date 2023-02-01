// +k8s:deepcopy-gen=package

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterWorkloadClassInstance ProjectWorkloadClassInstance

func (in *ClusterWorkloadClassInstance) NamespaceScoped() bool {
	return false
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterWorkloadClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterWorkloadClassInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectWorkloadClassInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Description       string              `json:"description,omitempty"`
	CPUScaler         float64             `json:"cpuScaler,omitempty"`
	Default           bool                `json:"default"`
	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	Memory            WorkloadClassMemory `json:"memory,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectWorkloadClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectWorkloadClassInstance `json:"items"`
}

type WorkloadClassMemory struct {
	Min     string   `json:"min,omitempty"`
	Max     string   `json:"max,omitempty"`
	Default string   `json:"default,omitempty"`
	Values  []string `json:"values,omitempty"`
}
