package v1

import "fmt"

type AppStatus struct {
	Containers map[string]ContainerStatus `json:"containers,omitempty"`
	Jobs       map[string]JobStatus       `json:"jobs,omitempty"`
	Volumes    map[string]VolumeStatus    `json:"volumes,omitempty"`
	Secrets    map[string]SecretStatus    `json:"secrets,omitempty"`
	Acorns     map[string]AcornStatus     `json:"acorns,omitempty"`
	Routers    map[string]RouterStatus    `json:"routers,omitempty"`
	Services   map[string]ServiceStatus   `json:"services,omitempty"`

	Endpoints []Endpoint `json:"endpoints,omitempty"`
	Stopped   bool       `json:"stopped,omitempty"`
	Completed bool       `json:"completed,omitempty"`
}

type DependencyNotFound struct {
	DependencyType DependencyType `json:"dependencyType,omitempty"`
	Name           string         `json:"name,omitempty"`
}

type ExpressionError struct {
	DependencyNotFound *DependencyNotFound `json:"dependencyNotFound,omitempty"`
	Expression         string              `json:"expression,omitempty"`
	Error              string              `json:"error,omitempty"`
}

func (e *ExpressionError) String() string {
	if e.DependencyNotFound == nil {
		return "error [" + e.Error + "] expression [" + e.Expression + "]"
	}
	if e.Expression != "" {
		return fmt.Sprintf("missing %s [%s] from expression [%s]", e.DependencyNotFound.DependencyType, e.DependencyNotFound.Name, e.Expression)
	}
	return fmt.Sprintf("missing %s [%s]", e.DependencyNotFound.DependencyType, e.DependencyNotFound.Name)
}

type ReplicasSummary struct {
	RunningCount           int
	MaxReplicaRestartCount int32
	TransitioningMessages  []string
	ErrorMessages          []string
}

type CommonStatus struct {
	Ready                 bool     `json:"ready,omitempty"`
	UpToDate              bool     `json:"upToDate,omitempty"`
	Defined               bool     `json:"defined,omitempty"`
	LinkOverride          string   `json:"linkOverride,omitempty"`
	TransitioningMessages []string `json:"transitioningMessages,omitempty"`
	ErrorMessages         []string `json:"errorMessages,omitempty"`
}

type AcornStatus struct {
	CommonStatus `json:",inline"`
	AcornName    string `json:"acornName,omitempty"`
}

func (in AcornStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type RouterStatus struct {
	CommonStatus `json:",inline"`
}

func (in RouterStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type ServiceStatus struct {
	CommonStatus      `json:",inline"`
	Default           bool              `json:"default,omitempty"`
	Ports             Ports             `json:"ports,omitempty"`
	Data              GenericMap        `json:"data,omitempty"`
	Secrets           []string          `json:"secrets,omitempty"`
	Address           string            `json:"address,omitempty"`
	Endpoint          string            `json:"endpoint,omitempty"`
	ServiceAcornName  string            `json:"serviceAcornName,omitempty"`
	ServiceAcornReady bool              `json:"serviceAcornReady,omitempty"`
	ExpressionErrors  []ExpressionError `json:"expressionErrors,omitempty"`
}

func (in ServiceStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type SecretStatus struct {
	CommonStatus        `json:",inline"`
	SecretName          string   `json:"secretName,omitempty"`
	JobName             string   `json:"jobName,omitempty"`
	JobReady            bool     `json:"jobReady,omitempty"`
	LookupErrors        []string `json:"lookupErrors,omitempty"`
	LookupTransitioning []string `json:"lookupTransitioning,omitempty"`
	DataKeys            []string `json:"dataKeys,omitempty"`
}

func (in SecretStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type ContainerStatus struct {
	CommonStatus           `json:",inline"`
	ReadyReplicaCount      int32                       `json:"readyCount,omitempty"`
	DesiredReplicaCount    int32                       `json:"readyDesiredCount,omitempty"`
	RunningReplicaCount    int32                       `json:"runningReplicaCount,omitempty"`
	UpToDateReplicaCount   int32                       `json:"upToDateCount,omitempty"`
	MaxReplicaRestartCount int32                       `json:"maxReplicaRestartCount,omitempty"`
	Dependencies           map[string]DependencyStatus `json:"dependencies,omitempty"`
	ExpressionErrors       []ExpressionError           `json:"expressionErrors,omitempty"`
}

func (in ContainerStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type JobStatus struct {
	CommonStatus         `json:",inline"`
	RunningCount         int                         `json:"runningCount,omitempty"`
	ErrorCount           int                         `json:"errorCount,omitempty"`
	CreateEventSucceeded bool                        `json:"createEventSucceeded,omitempty"`
	Dependencies         map[string]DependencyStatus `json:"dependencies,omitempty"`
	Skipped              bool                        `json:"skipped,omitempty"`
	ExpressionErrors     []ExpressionError           `json:"expressionErrors,omitempty"`
}

type DependencyStatus struct {
	Ready          bool           `json:"ready,omitempty"`
	Missing        bool           `json:"missing,omitempty"`
	DependencyType DependencyType `json:"serviceType,omitempty"`
}

type DependencyType string

const (
	DependencySecret    = DependencyType("secret")
	DependencyService   = DependencyType("service")
	DependencyJob       = DependencyType("job")
	DependencyContainer = DependencyType("container")
)

func (in JobStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type VolumeStatus struct {
	CommonStatus      `json:",inline"`
	VolumeName        string `json:"volumeName,omitempty"`
	StorageClassFound bool   `json:"storageClassFound,omitempty"`
	Bound             bool   `json:"bound,omitempty"`
}

func (in VolumeStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}
