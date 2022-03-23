// +k8s:deepcopy-gen=package

package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AppInstanceCondition string

var (
	AppInstanceConditionParsed  = "parsed"
	AppInstanceConditionPulled  = "pulled"
	AppInstanceConditionSecrets = "secrets"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   AppInstanceSpec   `json:"spec,omitempty"`
	Status AppInstanceStatus `json:"status,omitempty"`
}

type AppInstanceSpec struct {
	Image   string          `json:"image,omitempty"`
	Stop    *bool           `json:"stop,omitempty"`
	Volumes []VolumeBinding `json:"volumes,omitempty"`
	Secrets []SecretBinding `json:"secrets,omitempty"`
}

type SecretBinding struct {
	Secret        string `json:"secret,omitempty"`
	SecretRequest string `json:"secretRequest,omitempty"`
}

type VolumeBinding struct {
	Volume        string `json:"volume,omitempty"`
	VolumeRequest string `json:"volumeRequest,omitempty"`
}

type AppInstanceStatus struct {
	Namespace  string               `json:"namespace,omitempty"`
	AppImage   AppImage             `json:"appImage,omitempty"`
	AppSpec    AppSpec              `json:"appSpec,omitempty"`
	Conditions map[string]Condition `json:"conditions,omitempty"`
}

func (a *AppInstance) Conditions() *map[string]Condition {
	return &a.Status.Conditions
}
