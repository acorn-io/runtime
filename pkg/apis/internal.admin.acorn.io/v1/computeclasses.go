// +k8s:deepcopy-gen=package

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterComputeClassInstance ProjectComputeClassInstance

func (in *ClusterComputeClassInstance) NamespaceScoped() bool {
	return false
}

// EnsureRegion checks that the class supports the region. If it does not, then the region is added.
func (in *ClusterComputeClassInstance) EnsureRegion(region string) bool {
	for _, r := range in.SupportedRegions {
		if r == region {
			return true
		}
	}
	in.SupportedRegions = append(in.SupportedRegions, region)
	return true
}

func (in *ClusterComputeClassInstance) HasRegion(region string) bool {
	return slices.Contains(in.SupportedRegions, region)
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
	Description       string                       `json:"description,omitempty"`
	CPUScaler         float64                      `json:"cpuScaler,omitempty"`
	Default           bool                         `json:"default"`
	Affinity          *corev1.Affinity             `json:"affinity,omitempty"`
	Tolerations       []corev1.Toleration          `json:"tolerations,omitempty"`
	Memory            ComputeClassMemory           `json:"memory,omitempty"`
	SupportedRegions  []string                     `json:"supportedRegions,omitempty"`
	PriorityClassName string                       `json:"priorityClassName,omitempty"`
	RuntimeClassName  string                       `json:"runtimeClassName,omitempty"`
	Resources         *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectComputeClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectComputeClassInstance `json:"items"`
}

type ComputeClassMemory struct {
	Min           string   `json:"min,omitempty"`
	Max           string   `json:"max,omitempty"`
	Default       string   `json:"default,omitempty"`
	RequestScalar float64  `json:"requestScalar,omitempty"`
	Values        []string `json:"values,omitempty"`
}
