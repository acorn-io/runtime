package v1

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectVolumeClassInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	StorageClassName   string          `json:"storageClassName"`
	Description        string          `json:"description"`
	Default            bool            `json:"default,omitempty"`
	AllowedAccessModes v1.AccessModes  `json:"allowedAccessModes,omitempty"`
	Size               VolumeClassSize `json:"size,omitempty"`
	Inactive           bool            `json:"inactive,omitempty"`
	SupportedRegions   []string        `json:"supportedRegions,omitempty"`
}

type VolumeClassSize struct {
	Default v1.Quantity `json:"default,omitempty"`
	Min     v1.Quantity `json:"min,omitempty"`
	Max     v1.Quantity `json:"max,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectVolumeClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectVolumeClassInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterVolumeClassInstance ProjectVolumeClassInstance

func (c *ClusterVolumeClassInstance) NamespaceScoped() bool {
	return false
}

// EnsureRegion checks that the class supports the region. If it does not, then the region is added.
func (c *ClusterVolumeClassInstance) EnsureRegion(region string) bool {
	for _, r := range c.SupportedRegions {
		if r == region {
			return true
		}
	}
	c.SupportedRegions = append(c.SupportedRegions, region)
	return true
}

func (c *ClusterVolumeClassInstance) HasRegion(region string) bool {
	return slices.Contains(c.SupportedRegions, region)
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterVolumeClassInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterVolumeClassInstance `json:"items"`
}
