package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/strings/slices"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Spec              ProjectInstanceSpec   `json:"spec,omitempty"`
	Status            ProjectInstanceStatus `json:"status,omitempty"`
}

type ProjectInstanceSpec struct {
	DefaultRegion    string   `json:"defaultRegion,omitempty"`
	SupportedRegions []string `json:"supportedRegions,omitempty"`
}

type ProjectInstanceStatus struct {
	Namespace     string `json:"namespace,omitempty"`
	DefaultRegion string `json:"defaultRegion,omitempty"`
	// SupportedRegions on the status field should be an explicit list of supported regions.
	// That is, if the user specifies "*" for supported regions, then the status value should be the list of all regions.
	// This is to avoid having to make another call to explicitly list all regions.
	SupportedRegions []string `json:"supportedRegions,omitempty"`
}

func (in *ProjectInstance) NamespaceScoped() bool {
	return false
}

func (in *ProjectInstance) HasRegion(region string) bool {
	return region == "" || slices.Contains(in.Status.SupportedRegions, region)
}

func (in *ProjectInstance) GetRegion() string {
	return in.Status.DefaultRegion
}

func (in *ProjectInstance) GetSupportedRegions() []string {
	return in.Status.SupportedRegions
}

func (in *ProjectInstance) SetDefaultRegion(region string) {
	if in.Spec.DefaultRegion == "" && len(in.Spec.SupportedRegions) == 0 {
		in.Status.DefaultRegion = region
		in.Status.SupportedRegions = []string{"*"}
	} else {
		// Set the status values to the provided spec values.
		// The idea here is that internally, we only need to check the status values.
		in.Status.DefaultRegion = in.Spec.DefaultRegion
		in.Status.SupportedRegions = in.Spec.SupportedRegions
	}
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectInstance `json:"items"`
}
