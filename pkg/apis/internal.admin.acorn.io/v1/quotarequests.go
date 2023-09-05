package v1

import (
	"fmt"
	"strings"

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
	Region    string                `json:"region,omitempty"`
	Resources QuotaRequestResources `json:"resources,omitempty"`
}

type QuotaRequestInstanceStatus struct {
	ObservedGeneration int64                  `json:"observedGeneration,omitempty"`
	AllocatedResources QuotaRequestResources  `json:"allocatedResources,omitempty"`
	FailedResources    *QuotaRequestResources `json:"failedResources,omitempty"`
	Conditions         []v1.Condition         `json:"conditions,omitempty"`
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
type QuotaRequestResources struct {
	BaseResources `json:",inline"`
	Secrets       int `json:"secrets,omitempty"`
}

// Add will add the resources of another Resources struct into the current one.
func (current *QuotaRequestResources) Add(incoming QuotaRequestResources) {
	current.Secrets = Add(current.Secrets, incoming.Secrets)
	current.BaseResources.Add(incoming.BaseResources)
}

// Remove will remove the resources of another Resources struct from the current one. Calling remove
// will be a no-op for any resource values that are set to unlimited.
func (current *QuotaRequestResources) Remove(incoming QuotaRequestResources, all bool) {
	if all {
		current.Secrets = Sub(current.Secrets, incoming.Secrets)
	}
	current.BaseResources.Remove(incoming.BaseResources, all)
}

// Fits will check if a group of resources will be able to contain
// another group of resources. If the resources are not able to fit,
// an aggregated error will be returned with all exceeded resources.
// If the current resources defines unlimited, then it will always fit.
func (current *QuotaRequestResources) Fits(incoming QuotaRequestResources) error {
	exceededResources := []string{}
	Fits(exceededResources, "Secrets", current.Secrets, incoming.Secrets)
	// Build an aggregated error message for the exceeded resources
	if len(exceededResources) > 0 {
		return fmt.Errorf("%w: %s", ErrExceededResources, strings.Join(exceededResources, ", "))
	}
	return current.BaseResources.Fits(incoming.BaseResources)
}

// NonEmptyString will return a string representation of the non-empty
// Resources within the struct.
func (current *QuotaRequestResources) NonEmptyString() string {
	resources := ResourceToString("Secrets", current.Secrets)
	return strings.Join([]string{current.BaseResources.NonEmptyString(), strings.Join(resources, ", ")}, ", ")
}

// Equals will check if the current Resources struct is equal to another. This is useful
// to avoid needing to do a deep equal on the entire struct.
func (current *QuotaRequestResources) Equals(incoming QuotaRequestResources) bool {
	return current.BaseResources.Equals(incoming.BaseResources) &&
		current.Secrets == incoming.Secrets
}
