package v1

import (
	internaladminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ComputeClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Memory           ComputeClassMemory           `json:"memory,omitempty"`
	Resources        *corev1.ResourceRequirements `json:"resources,omitempty"`
	Description      string                       `json:"description,omitempty"`
	Default          bool                         `json:"default"`
	SupportedRegions []string                     `json:"supportedRegions,omitempty"`
}

type ComputeClassMemory struct {
	Min     string   `json:"min,omitempty"`
	Max     string   `json:"max,omitempty"`
	Default string   `json:"default,omitempty"`
	Values  []string `json:"values,omitempty"`
}

// ComputeClassMemoryFromInternalAdmin casts an internal admin ComputeClassMemory object to an api ComputeClassMemory object
// This is done to hide the requestScaler value from api endpoints
func ComputeClassMemoryFromInternalAdmin(memory internaladminv1.ComputeClassMemory) ComputeClassMemory {
	return ComputeClassMemory{
		Min:     memory.Min,
		Max:     memory.Max,
		Default: memory.Default,
		Values:  memory.Values,
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ComputeClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComputeClass `json:"items"`
}
