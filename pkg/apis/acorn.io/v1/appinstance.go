// +k8s:deepcopy-gen=package

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppInstanceCondition string

var (
	AppInstanceConditionDefined    = "defined"
	AppInstanceConditionParsed     = "parsed"
	AppInstanceConditionPulled     = "pulled"
	AppInstanceConditionSecrets    = "secrets"
	AppInstanceConditionContainers = "containers"
	AppInstanceConditionJobs       = "jobs"
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
	Image            string            `json:"image,omitempty"`
	Stop             *bool             `json:"stop,omitempty"`
	ReattachVolumes  *bool             `json:"reattachVolumes,omitempty"`
	ReattachSecrets  *bool             `json:"reattachSecrets,omitempty"`
	Volumes          []VolumeBinding   `json:"volumes,omitempty"`
	Secrets          []SecretBinding   `json:"secrets,omitempty"`
	Endpoints        []EndpointBinding `json:"endpoints,omitempty"`
	Services         []ServiceBinding  `json:"services,omitempty"`
	PublishProtocols []Protocol        `json:"publishProtocols,omitempty"`
	Ports            []PortBinding     `json:"ports,omitempty"`
	DeployParams     GenericMap        `json:"deployParams,omitempty"`
}

type ServiceBinding struct {
	Target  string `json:"target,omitempty"`
	Service string `json:"service,omitempty"`
}

type EndpointBinding struct {
	Target   string `json:"target,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type SecretBinding struct {
	Secret        string `json:"secret,omitempty"`
	SecretRequest string `json:"secretRequest,omitempty"`
}

type VolumeBinding struct {
	Volume        string            `json:"volume,omitempty"`
	VolumeRequest string            `json:"volumeRequest,omitempty"`
	Capacity      resource.Quantity `json:"capacity,omitempty"`
	AccessModes   []AccessMode      `json:"accessModes,omitempty"`
	Class         string            `json:"class,omitempty"`
}

type ContainerStatus struct {
	Ready        int32 `json:"ready,omitempty"`
	ReadyDesired int32 `json:"readyDesired,omitempty"`
	UpToDate     int32 `json:"upToDate,omitempty"`
	RestartCount int32 `json:"restartCount,omitempty"`
}

type JobStatus struct {
	Succeed bool   `json:"succeed,omitempty"`
	Failed  bool   `json:"failed,omitempty"`
	Running bool   `json:"running,omitempty"`
	Message string `json:"message,omitempty"`
}

type AppColumns struct {
	Healthy   string `json:"healthy,omitempty" column:"name=Healthy,jsonpath=.status.columns.healthy"`
	UpToDate  string `json:"upToDate,omitempty" column:"name=Up-To-Date,jsonpath=.status.columns.upToDate"`
	Message   string `json:"message,omitempty" column:"name=Message,jsonpath=.status.columns.message"`
	Endpoints string `json:"endpoints,omitempty" column:"name=Endpoints,jsonpath=.status.columns.endpoints"`
	Created   string `json:"created,omitempty" column:"name=Created,jsonpath=.metadata.creationTimestamp"`
}

type AppInstanceStatus struct {
	Columns         AppColumns                 `json:"columns,omitempty"`
	ContainerStatus map[string]ContainerStatus `json:"containerStatus,omitempty"`
	JobsStatus      map[string]JobStatus       `json:"jobsStatus,omitempty"`
	Stopped         bool                       `json:"stopped,omitempty"`
	Namespace       string                     `json:"namespace,omitempty"`
	AppImage        AppImage                   `json:"appImage,omitempty"`
	AppSpec         AppSpec                    `json:"appSpec,omitempty"`
	Conditions      map[string]Condition       `json:"conditions,omitempty"`
	Endpoints       []Endpoint                 `json:"endpoints,omitempty"`
}

type Endpoint struct {
	Target     string   `json:"target,omitempty"`
	TargetPort int32    `json:"targetPort,omitempty"`
	Address    string   `json:"address,omitempty"`
	Protocol   Protocol `json:"protocol,omitempty"`
}

func (a *AppInstance) Conditions() *map[string]Condition {
	return &a.Status.Conditions
}
