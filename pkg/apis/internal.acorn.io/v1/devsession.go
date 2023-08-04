package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DevSessionInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DevSessionInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DevSessionInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   DevSessionInstanceSpec   `json:"spec,omitempty"`
	Status DevSessionInstanceStatus `json:"status,omitempty"`
}

func (in *DevSessionInstance) HasRegion(region string) bool {
	return in.Spec.Region == region
}

type DevSessionInstanceSpec struct {
	Region                string                   `json:"region,omitempty"`
	Client                DevSessionInstanceClient `json:"client,omitempty"`
	SessionTimeoutSeconds int32                    `json:"sessionTimeoutSeconds,omitempty"`
	SessionStartTime      metav1.Time              `json:"sessionStartTime,omitempty"`
	SessionRenewTime      metav1.Time              `json:"sessionRenewTime,omitempty"`
	SpecOverride          *AppInstanceSpec         `json:"specOverride,omitempty"`
}

type DevSessionInstanceStatus struct {
	Expired    bool        `json:"expired,omitempty"`
	Conditions []Condition `json:"conditions,omitempty"`
}

type DevSessionInstanceClient struct {
	Hostname    string                `json:"hostname,omitempty"`
	ImageSource DevSessionImageSource `json:"imageSource,omitempty"`
}

type DevSessionImageSource struct {
	Image string `json:"image,omitempty"`
	File  string `json:"file,omitempty"`
}

type DevSessionInstanceExpireAction struct {
	Stop bool `json:"stop,omitempty"`
}
