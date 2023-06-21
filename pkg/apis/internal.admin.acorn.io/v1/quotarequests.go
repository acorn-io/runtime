package v1

import (
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const QuotaRequestCondition = "quota-request"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QuotaRequestInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaRequestInstanceSpec   `json:"spec,omitempty"`
	Status QuotaRequestInstanceStatus `json:"status,omitempty"`
}

// EnsureRegion checks or sets the region of a QuotaRequstInstance.
// If a QuotaRequstInstance's region is unset, EnsureRegion sets it to the given region and returns true.
// Otherwise, it returns true if and only if the Volume belongs to the given region.
func (in *QuotaRequestInstance) EnsureRegion(region string) bool {
	// If the region of a QuotaRequstInstance is not set, then it hasn't been synced yet. In this case, we assume that the QuotaRequstInstance is in
	// the same region as the app, and return true.
	if in.Spec.Region == "" {
		in.Spec.Region = region
	}

	return in.Spec.Region == region
}

func (in *QuotaRequestInstance) HasRegion(region string) bool {
	return in.Spec.Region == region
}

func (in *QuotaRequestInstance) GetRegion() string {
	return in.Spec.Region
}

type QuotaRequestInstanceSpec struct {
	Region    string    `json:"region,omitempty"`
	Resources Resources `json:"resources,omitempty"`
}

type QuotaRequestInstanceStatus struct {
	ObservedGeneration int64          `json:"observedGeneration,omitempty"`
	AllocatedResources Resources      `json:"allocatedResources,omitempty"`
	FailedResources    *Resources     `json:"failedResources,omitempty"`
	Conditions         []v1.Condition `json:"conditions,omitempty"`
}

func (in *QuotaRequestInstanceStatus) Condition(name string) v1.Condition {
	for _, cond := range in.Conditions {
		if cond.Type == name {
			return cond
		}
	}
	return v1.Condition{}
}

func (in *QuotaRequestInstance) Conditions() *[]v1.Condition {
	return &in.Status.Conditions
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type QuotaRequestInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []QuotaRequestInstance `json:"items"`
}
