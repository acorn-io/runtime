package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceInstance `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec ServiceInstanceSpec `json:"spec,omitempty"`
}

type ServiceInstanceSpec struct {
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Default         bool              `json:"default"`
	External        string            `json:"external,omitempty"`
	Address         string            `json:"address,omitempty"`
	Ports           Ports             `json:"ports,omitempty"`
	Container       string            `json:"container,omitempty"`
	ContainerLabels map[string]string `json:"containerLabels,omitempty"`
	Secrets         []string          `json:"secrets,omitempty"`
	Attributes      GenericMap        `json:"attributes,omitempty"`
	Destroy         *Container        `json:"destroy,omitempty"`
}
