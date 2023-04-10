// +k8s:deepcopy-gen=package

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppInstanceCondition string

var (
	AppInstanceConditionDefined      = "defined"
	AppInstanceConditionDefaults     = "defaults"
	AppInstanceConditionScheduling   = "scheduling"
	AppInstanceConditionNamespace    = "namespace"
	AppInstanceConditionParsed       = "parsed"
	AppInstanceConditionController   = "controller"
	AppInstanceConditionPulled       = "image-pull"
	AppInstanceConditionSecrets      = "secrets"
	AppInstanceConditionContainers   = "containers"
	AppInstanceConditionJobs         = "jobs"
	AppInstanceConditionAcorns       = "acorns"
	AppInstanceConditionReady        = "Ready"
	AppInstanceConditionVolumes      = "volumes"
	AppInstanceConditionImageAllowed = "image-allowed"
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

func (in *AppInstance) HasRegion(region string) bool {
	return in.Status.Defaults.Region == region || in.Spec.Region == region
}

func (in *AppInstance) GetRegion() string {
	if in.Spec.Region != "" {
		return in.Spec.Region
	}
	return in.Status.Defaults.Region
}

func (in *AppInstance) SetDefaultRegion(region string) {
	if in.Spec.Region == "" {
		in.Status.Defaults.Region = region
	} else {
		in.Status.Defaults.Region = ""
	}
}

func (in *AppInstance) ShortID() string {
	if len(in.UID) > 11 {
		return string(in.UID[:12])
	}
	return string(in.UID)
}

type PublishMode string

const (
	PublishModeAll     = PublishMode("all")
	PublishModeNone    = PublishMode("none")
	PublishModeDefined = PublishMode("defined")
)

type AppInstanceSpec struct {
	Region              string           `json:"region,omitempty"`
	Labels              []ScopedLabel    `json:"labels,omitempty"`
	Annotations         []ScopedLabel    `json:"annotations,omitempty"`
	Image               string           `json:"image,omitempty"`
	Stop                *bool            `json:"stop,omitempty"`
	DevMode             *bool            `json:"devMode,omitempty"`
	Profiles            []string         `json:"profiles,omitempty"`
	Volumes             []VolumeBinding  `json:"volumes,omitempty"`
	Secrets             []SecretBinding  `json:"secrets,omitempty"`
	Environment         []NameValue      `json:"environment,omitempty"`
	PublishMode         PublishMode      `json:"publishMode,omitempty"`
	TargetNamespace     string           `json:"targetNamespace,omitempty"`
	Links               []ServiceBinding `json:"services,omitempty"`
	Publish             []PortBinding    `json:"ports,omitempty"`
	DeployArgs          GenericMap       `json:"deployArgs,omitempty"`
	Permissions         []Permissions    `json:"permissions,omitempty"`
	AutoUpgrade         *bool            `json:"autoUpgrade,omitempty"`
	NotifyUpgrade       *bool            `json:"notifyUpgrade,omitempty"`
	AutoUpgradeInterval string           `json:"autoUpgradeInterval,omitempty"`
	ComputeClasses      ComputeClassMap  `json:"computeClass,omitempty"`
	Memory              MemoryMap        `json:"memory,omitempty"`
}

func (in *AppInstanceSpec) GetStopped() bool {
	return in.Stop != nil && *in.Stop
}

func (in *AppInstanceSpec) GetAutoUpgrade() bool {
	return in.AutoUpgrade != nil && *in.AutoUpgrade
}

func (in *AppInstanceSpec) GetNotifyUpgrade() bool {
	return in.NotifyUpgrade != nil && *in.NotifyUpgrade
}

func (in *AppInstanceSpec) GetDevMode() bool {
	return in.DevMode != nil && *in.DevMode
}

func (in *AppInstanceSpec) GetProfiles() []string {
	if in.GetDevMode() {
		found := false
		for _, profile := range in.Profiles {
			if profile == "dev?" {
				found = true
				break
			}
		}
		if !found {
			return append([]string{"dev?"}, in.Profiles...)
		}
	}
	return in.Profiles
}

type ServiceBindings []ServiceBinding

type ServiceBinding struct {
	Target  string `json:"target,omitempty"`
	Service string `json:"service,omitempty"`
}

type SecretBindings []SecretBinding

type SecretBinding struct {
	Secret string `json:"secret,omitempty"`
	Target string `json:"target,omitempty"`
}

type Quantity string

type VolumeBindings []VolumeBinding

type VolumeBinding struct {
	Volume      string      `json:"volume,omitempty"`
	Target      string      `json:"target,omitempty"`
	Size        Quantity    `json:"size,omitempty"`
	AccessModes AccessModes `json:"accessModes,omitempty"`
	Class       string      `json:"class,omitempty"`
}

type ContainerStatus struct {
	Ready        int32 `json:"ready,omitempty"`
	ReadyDesired int32 `json:"readyDesired,omitempty"`
	UpToDate     int32 `json:"upToDate,omitempty"`
	RestartCount int32 `json:"restartCount,omitempty"`
	Created      bool  `json:"created,omitempty"`
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
	ObservedGeneration     int64                      `json:"observedGeneration,omitempty"`
	Columns                AppColumns                 `json:"columns,omitempty"`
	ContainerStatus        map[string]ContainerStatus `json:"containerStatus,omitempty"`
	JobsStatus             map[string]JobStatus       `json:"jobsStatus,omitempty"`
	AcornStatus            map[string]AcornStatus     `json:"acornStatus,omitempty"`
	Ready                  bool                       `json:"ready,omitempty"`
	Stopped                bool                       `json:"stopped,omitempty"`
	Namespace              string                     `json:"namespace,omitempty"`
	AppImage               AppImage                   `json:"appImage,omitempty"`
	AvailableAppImage      string                     `json:"availableAppImage,omitempty"`
	ConfirmUpgradeAppImage string                     `json:"confirmUpgradeAppImage,omitempty"`
	AppSpec                AppSpec                    `json:"appSpec,omitempty"`
	Scheduling             map[string]Scheduling      `json:"scheduling,omitempty"`
	Conditions             []Condition                `json:"conditions,omitempty"`
	Endpoints              []Endpoint                 `json:"endpoints,omitempty"`
	Defaults               Defaults                   `json:"defaults,omitempty"`
}

type Defaults struct {
	Volumes map[string]VolumeDefault `json:"volumes,omitempty"`
	Memory  map[string]*int64        `json:"memory,omitempty"`
	Region  string                   `json:"region,omitempty"`
}

type VolumeDefault struct {
	Class       string      `json:"class,omitempty"`
	Size        Quantity    `json:"size,omitempty"`
	AccessModes AccessModes `json:"accessModes,omitempty"`
}

type Scheduling struct {
	Requirements corev1.ResourceRequirements `json:"requirements,omitempty"`
	Affinity     *corev1.Affinity            `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration         `json:"tolerations,omitempty"`
}

type AcornStatus struct {
	ContainerStatus map[string]ContainerStatus `json:"containerStatus,omitempty"`
	JobsStatus      map[string]JobStatus       `json:"jobsStatus,omitempty"`
	AcornStatus     map[string]AcornStatus     `json:"acornStatus,omitempty"`
	Stopped         bool                       `json:"stopped,omitempty"`
	Ready           bool                       `json:"ready,omitempty"`
}

type Endpoint struct {
	Target          string          `json:"target,omitempty"`
	TargetPort      int32           `json:"targetPort,omitempty"`
	Address         string          `json:"address,omitempty"`
	Protocol        Protocol        `json:"protocol,omitempty"`
	PublishProtocol PublishProtocol `json:"publishProtocol,omitempty"`
	Pending         bool            `json:"pending,omitempty"`
}

func (in *AppInstanceStatus) Condition(name string) Condition {
	for _, cond := range in.Conditions {
		if cond.Type == name {
			return cond
		}
	}
	return Condition{}
}

func (in *AppInstance) Conditions() *[]Condition {
	return &in.Status.Conditions
}
