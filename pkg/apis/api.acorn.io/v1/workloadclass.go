package v1

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WorkloadClass struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Memory      v1.WorkloadClassMemory `json:"memory,omitempty"`
	Description string                 `json:"description,omitempty"`
	Default     bool                   `json:"default"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type WorkloadClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WorkloadClass `json:"items"`
}
