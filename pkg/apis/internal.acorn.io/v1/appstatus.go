package v1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AppStatus struct {
	Permissions []Permissions `json:"permissions,omitempty"`

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
	SubKey         string         `json:"subKey,omitempty"`
}

type ExpressionError struct {
	DependencyNotFound *DependencyNotFound `json:"dependencyNotFound,omitempty"`
	Expression         string              `json:"expression,omitempty"`
	Error              string              `json:"error,omitempty"`
}

// IsMissingDependencyError indicates this error is because of a missing resource and can typically
// be treated as a transient error
func (e *ExpressionError) IsMissingDependencyError() bool {
	return e.DependencyNotFound != nil && e.DependencyNotFound.SubKey == ""
}

func (e *ExpressionError) String() string {
	if e.DependencyNotFound == nil {
		return "error [" + e.Error + "] expression [" + e.Expression + "]"
	}
	prefix := ""
	suffix := ""
	if e.DependencyNotFound.SubKey != "" {
		prefix = fmt.Sprintf("key [%s] in ", e.DependencyNotFound.SubKey)
	}
	if e.Expression != "" {
		suffix = fmt.Sprintf(" from expression [%s]", e.Expression)
	}

	return fmt.Sprintf("missing %s%s [%s]%s", prefix, e.DependencyNotFound.DependencyType, e.DependencyNotFound.Name, suffix)
}

type ReplicasSummary struct {
	RunningCount           int
	MaxReplicaRestartCount int32
	TransitioningMessages  []string
	ErrorMessages          []string
}

type CommonSummary struct {
	State                 string   `json:"state,omitempty"`
	Messages              []string `json:"messages,omitempty"`
	TransitioningMessages []string `json:"transitioningMessages,omitempty"`
	ErrorMessages         []string `json:"errorMessages,omitempty"`
}

type CommonStatus struct {
	State                 string   `json:"state,omitempty"`
	Ready                 bool     `json:"ready,omitempty"`
	UpToDate              bool     `json:"upToDate,omitempty"`
	Defined               bool     `json:"defined,omitempty"`
	LinkOverride          string   `json:"linkOverride,omitempty"`
	Messages              []string `json:"messages,omitempty"`
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
	CommonStatus   `json:",inline"`
	MissingTargets []string `json:"missingTargets,omitempty"`
}

func (in RouterStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}

type ServiceStatus struct {
	CommonStatus               `json:",inline"`
	Default                    bool              `json:"default,omitempty"`
	Ports                      Ports             `json:"ports,omitempty"`
	Data                       *GenericMap       `json:"data,omitempty"`
	Consumer                   *ServiceConsumer  `json:"consumer,omitempty"`
	Secrets                    []string          `json:"secrets,omitempty"`
	Address                    string            `json:"address,omitempty"`
	Endpoint                   string            `json:"endpoint,omitempty"`
	ServiceAcornName           string            `json:"serviceAcornName,omitempty"`
	ServiceAcornReady          bool              `json:"serviceAcornReady,omitempty"`
	MissingConsumerPermissions []Permissions     `json:"missingConsumerPermissions,omitempty"`
	ExpressionErrors           []ExpressionError `json:"expressionErrors,omitempty"`
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
	Schedule             string                      `json:"schedule,omitempty"`
	JobName              string                      `json:"name,omitempty"`
	JobNamespace         string                      `json:"namespace,omitempty"`
	CreationTime         *metav1.Time                `json:"creationTime,omitempty"`
	StartTime            *metav1.Time                `json:"startTime,omitempty"`
	CompletionTime       *metav1.Time                `json:"completionTime,omitempty"`
	LastRun              *metav1.Time                `json:"lastRun,omitempty"`
	NextRun              *metav1.Time                `json:"nextRun,omitempty"`
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
	Unused            bool   `json:"unused,omitempty"`
}

func (in VolumeStatus) GetCommonStatus() CommonStatus {
	return in.CommonStatus
}
