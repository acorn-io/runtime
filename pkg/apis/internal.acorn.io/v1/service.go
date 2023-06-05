package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceInstance `json:"items"`
}

type ServiceInstanceCondition string

var (
	ServiceInstanceConditionDefined = "defined"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ServiceInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec   ServiceInstanceSpec   `json:"spec,omitempty"`
	Status ServiceInstanceStatus `json:"status,omitempty"`
}

type ServiceInstanceStatus struct {
	Conditions []Condition `json:"conditions,omitempty"`
	Endpoints  []Endpoint  `json:"endpoints,omitempty"`
	HasService bool        `json:"hasService,omitempty"`
}

func (in *ServiceInstance) ShortID() string {
	if len(in.UID) > 11 {
		return string(in.UID[:12])
	}
	return string(in.UID)
}

type ServiceInstanceSpec struct {
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Default         bool              `json:"default"`
	External        string            `json:"external,omitempty"`
	Alias           string            `json:"alias,omitempty"`
	Address         string            `json:"address,omitempty"`
	Ports           Ports             `json:"ports,omitempty"`
	Container       string            `json:"container,omitempty"`
	Job             string            `json:"job,omitempty"`
	ContainerLabels map[string]string `json:"containerLabels,omitempty"`
	Secrets         []string          `json:"secrets,omitempty"`
	Data            GenericMap        `json:"data,omitempty"`

	// Fields from app
	AppName      string        `json:"appName,omitempty"`
	AppNamespace string        `json:"appNamespace,omitempty"`
	Routes       []Route       `json:"routes,omitempty"`
	PublishMode  PublishMode   `json:"publishMode,omitempty"`
	Publish      []PortPublish `json:"publish,omitempty"`
}

type PortPublish struct {
	Port       int32    `json:"port,omitempty"`
	Protocol   Protocol `json:"protocol,omitempty"`
	Hostname   string   `json:"hostname,omitempty"`
	TargetPort int32    `json:"targetPort,omitempty"`
}

func (in PortPublish) Complete() PortPublish {
	if in.Hostname != "" && in.Protocol == "" {
		in.Protocol = ProtocolHTTP
	}
	if in.Hostname != "" && in.Protocol != ProtocolHTTP {
		in.Hostname = ""
	}
	return in
}
