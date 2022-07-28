package v1

import (
	"fmt"
	"strings"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	VolumeRequestTypeEphemeral = "ephemeral"

	AccessModeReadWriteMany AccessMode = "readWriteMany"
	AccessModeReadWriteOnce AccessMode = "readWriteOnce"
	AccessModeReadOnlyMany  AccessMode = "readOnlyMany"
)

type AccessMode string

const (
	ChangeTypeRedeploy = "redeploy"
	ChangeTypeOnAction = "noAction"
)

type ChangeType string

type AcornBuild struct {
	OriginalImage string     `json:"originalImage,omitempty"`
	Context       string     `json:"context,omitempty"`
	Acornfile     string     `json:"acornfile,omitempty"`
	BuildArgs     GenericMap `json:"buildArgs,omitempty"`
}

type Build struct {
	Context     string            `json:"context,omitempty"`
	Dockerfile  string            `json:"dockerfile,omitempty"`
	Target      string            `json:"target,omitempty"`
	BaseImage   string            `json:"baseImage,omitempty"`
	ContextDirs map[string]string `json:"contextDirs,omitempty"`
	BuildArgs   map[string]string `json:"buildArgs,omitempty"`
}

func (in Build) BaseBuild() Build {
	return Build{
		Context:    in.Context,
		Dockerfile: in.Dockerfile,
		Target:     in.Target,
	}
}

type Protocol string

var (
	ProtocolTCP  = Protocol("tcp")
	ProtocolUDP  = Protocol("udp")
	ProtocolHTTP = Protocol("http")
)

type PortBinding struct {
	Expose            bool     `json:"expose,omitempty"`
	Port              int32    `json:"port,omitempty"`
	Protocol          Protocol `json:"protocol,omitempty"`
	Publish           bool     `json:"publish,omitempty"`
	ServiceName       string   `json:"serviceName,omitempty"`
	TargetPort        int32    `json:"targetPort,omitempty"`
	TargetServiceName string   `json:"targetServiceName,omitempty"`
}

func (in PortBinding) Complete(serviceName string) PortBinding {
	if in.ServiceName == "" {
		in.ServiceName = serviceName
	}
	if in.TargetPort == 0 {
		in.TargetPort = in.Port
	}
	return in
}

type PortDef struct {
	Expose      bool     `json:"expose,omitempty"`
	Port        int32    `json:"port,omitempty"`
	Protocol    Protocol `json:"protocol,omitempty"`
	Publish     bool     `json:"publish,omitempty"`
	ServiceName string   `json:"serviceName,omitempty"`
	TargetPort  int32    `json:"targetPort,omitempty"`
	// TargetServiceName is only used in portDefs for acorns, not containers
	TargetServiceName string `json:"targetServiceName,omitempty"`
}

func (in PortDef) String() string {
	in = in.Complete(in.ServiceName)
	buf := &strings.Builder{}
	if in.ServiceName != "" {
		buf.WriteString(in.ServiceName)
	}
	if in.Port != in.TargetPort {
		if buf.Len() > 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprint(in.Port))
	}
	if in.TargetServiceName != "" {
		if buf.Len() > 0 {
			buf.WriteString(":")
		}
		buf.WriteString(in.TargetServiceName)
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprint(in.TargetPort))
	buf.WriteString("/")
	buf.WriteString(string(in.Protocol))

	if in.Expose {
		buf.WriteString(",expose")
	}
	if in.Publish {
		buf.WriteString(",publish")
	}
	return buf.String()
}

func (in PortDef) Complete(serviceName string) PortDef {
	if in.ServiceName == "" && serviceName != "" {
		in.ServiceName = serviceName
	}
	if in.TargetPort == 0 {
		in.TargetPort = in.Port
	}
	if in.Port == 0 {
		in.Port = in.TargetPort
	}
	if in.Protocol == "" {
		in.Protocol = ProtocolTCP
	}
	return in
}

type FileSecret struct {
	Name     string     `json:"name,omitempty"`
	Key      string     `json:"key,omitempty"`
	OnChange ChangeType `json:"onChange,omitempty"`
}

type File struct {
	Mode    string     `json:"mode,omitempty"`
	Content string     `json:"content,omitempty"`
	Secret  FileSecret `json:"secret,omitempty"`
}

type VolumeSecretMount struct {
	Name     string     `json:"name,omitempty"`
	OnChange ChangeType `json:"onChange,omitempty"`
}

type VolumeMount struct {
	Volume     string            `json:"volume,omitempty"`
	SubPath    string            `json:"subPath,omitempty"`
	ContextDir string            `json:"contextDir,omitempty"`
	Secret     VolumeSecretMount `json:"secret,omitempty"`
}

type EnvVar struct {
	Name   string       `json:"name,omitempty"`
	Value  string       `json:"value,omitempty"`
	Secret EnvSecretVal `json:"secret,omitempty"`
}

type EnvSecretVal struct {
	Name     string     `json:"name,omitempty"`
	Key      string     `json:"key,omitempty"`
	OnChange ChangeType `json:"onChange,omitempty"`
}

type Alias struct {
	Name string `json:"name,omitempty"`
}

type ExecProbe struct {
	Command []string `json:"command,omitempty"`
}

type TCPProbe struct {
	URL string `json:"url,omitempty"`
}

type HTTPProbe struct {
	URL     string            `json:"url,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type ProbeType string

const (
	ReadinessProbeType ProbeType = "readiness"
	LivenessProbeType  ProbeType = "liveness"
	StartupProbeType   ProbeType = "startup"
)

type Probe struct {
	Type                ProbeType  `json:"type,omitempty"`
	Exec                *ExecProbe `json:"exec,omitempty"`
	HTTP                *HTTPProbe `json:"http,omitempty"`
	TCP                 *TCPProbe  `json:"tcp,omitempty"`
	InitialDelaySeconds int32      `json:"initialDelaySeconds,omitempty"`
	TimeoutSeconds      int32      `json:"timeoutSeconds,omitempty"`
	PeriodSeconds       int32      `json:"periodSeconds,omitempty"`
	SuccessThreshold    int32      `json:"successThreshold,omitempty"`
	FailureThreshold    int32      `json:"failureThreshold,omitempty"`
}

type Dependency struct {
	TargetName string `json:"targetName,omitempty"`
}

type Permissions struct {
	Rules        []rbacv1.PolicyRule `json:"rules,omitempty"`
	ClusterRules []rbacv1.PolicyRule `json:"clusterRules,omitempty"`
}

func (in *Permissions) HasRules() bool {
	if in == nil {
		return false
	}
	return len(in.ClusterRules) > 0 || len(in.Rules) > 0
}

func (in *Permissions) Get() Permissions {
	if in == nil {
		return Permissions{}
	}
	return *in
}

type Container struct {
	Dirs         map[string]VolumeMount `json:"dirs,omitempty"`
	Files        map[string]File        `json:"files,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Build        *Build                 `json:"build,omitempty"`
	Command      []string               `json:"command,omitempty"`
	Interactive  bool                   `json:"interactive,omitempty"`
	Entrypoint   []string               `json:"entrypoint,omitempty"`
	Environment  []EnvVar               `json:"environment,omitempty"`
	WorkingDir   string                 `json:"workingDir,omitempty"`
	Ports        []PortDef              `json:"ports,omitempty"`
	Probes       []Probe                `json:"probes,omitempty"`
	Dependencies []Dependency           `json:"dependencies,omitempty"`
	Permissions  *Permissions           `json:"permissions,omitempty"`

	// Scale is only available on containers, not sidecars or jobs
	Scale *int32 `json:"scale,omitempty"`

	// Schedule is only available on jobs
	Schedule string `json:"schedule,omitempty"`

	// Init is only available on sidecars
	Init bool `json:"init,omitempty"`

	// Sidecars are not available on sidecars
	Sidecars map[string]Container `json:"sidecars,omitempty"`
}

type Image struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type AppSpec struct {
	Containers map[string]Container     `json:"containers,omitempty"`
	Jobs       map[string]Container     `json:"jobs,omitempty"`
	Images     map[string]Image         `json:"images,omitempty"`
	Volumes    map[string]VolumeRequest `json:"volumes,omitempty"`
	Secrets    map[string]Secret        `json:"secrets,omitempty"`
	Acorns     map[string]Acorn         `json:"acorns,omitempty"`
}

type Acorn struct {
	Image        string              `json:"image,omitempty"`
	Build        *AcornBuild         `json:"build,omitempty"`
	DeployArgs   GenericMap          `json:"deployArgs,omitempty"`
	Ports        []PortDef           `json:"ports,omitempty"`
	Secrets      []SecretBinding     `json:"secrets,omitempty"`
	Volumes      []VolumeBinding     `json:"volumes,omitempty"`
	Services     []ServiceBinding    `json:"services,omitempty"`
	Roles        []rbacv1.PolicyRule `json:"roles,omitempty"`
	ClusterRoles []rbacv1.PolicyRule `json:"clusterRoles,omitempty"`
	Permissions  *Permissions        `json:"permissions,omitempty"`
}

type Secret struct {
	Type   string            `json:"type,omitempty"`
	Params GenericMap        `json:"params,omitempty"`
	Data   map[string]string `json:"data,omitempty"`
}

type VolumeRequest struct {
	Class       string       `json:"class,omitempty"`
	Size        int64        `json:"size,omitempty"`
	ContextPath string       `json:"contextPath,omitempty"`
	AccessModes []AccessMode `json:"accessModes,omitempty"`
}
