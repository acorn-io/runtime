package v1

import (
	"fmt"
	"sort"
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
	ChangeTypeNoAction = "noAction"
)

type PathType string

const (
	PathTypeExact  PathType = "exact"
	PathTypePrefix PathType = "prefix"
)

type ChangeType string

type AcornBuild struct {
	OriginalImage string     `json:"originalImage,omitempty"`
	Context       string     `json:"context,omitempty"`
	Acornfile     string     `json:"acornfile,omitempty"`
	BuildArgs     GenericMap `json:"buildArgs,omitempty"`
}

type Build struct {
	Context            string            `json:"context,omitempty"`
	Dockerfile         string            `json:"dockerfile,omitempty"`
	DockerfileContents string            `json:"dockerfileContents,omitempty"`
	Target             string            `json:"target,omitempty"`
	BaseImage          string            `json:"baseImage,omitempty"`
	ContextDirs        map[string]string `json:"contextDirs,omitempty"`
	BuildArgs          map[string]string `json:"buildArgs,omitempty"`
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

type PublishProtocol string

var (
	PublishProtocolTCP   = PublishProtocol("tcp")
	PublishProtocolUDP   = PublishProtocol("udp")
	PublishProtocolHTTP  = PublishProtocol("http")
	PublishProtocolHTTPS = PublishProtocol("https")
)

type PortBindings []PortBinding

type PortBinding struct {
	Port     int32    `json:"port,omitempty"`
	Protocol Protocol `json:"protocol,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	// Deprecated Use Hostname instead
	ZZ_ServiceName    string `json:"serviceName,omitempty"`
	TargetPort        int32  `json:"targetPort,omitempty"`
	TargetServiceName string `json:"targetServiceName,omitempty"`
}

func (in PortBinding) Complete() PortBinding {
	if in.Hostname != "" && in.Protocol == "" {
		in.Protocol = ProtocolHTTP
	}
	if in.Hostname != "" && in.Protocol != ProtocolHTTP {
		in.Hostname = ""
	}
	return in
}

func (in PortDef) FormatString(serviceName string) string {
	//in = in.Complete(in.ServiceName)
	buf := &strings.Builder{}
	if serviceName != "" {
		buf.WriteString(serviceName)
	}
	if in.Port != in.TargetPort {
		if buf.Len() > 0 {
			buf.WriteString(":")
		}
		buf.WriteString(fmt.Sprint(in.Port))
	}
	if buf.Len() > 0 {
		buf.WriteString(":")
	}
	buf.WriteString(fmt.Sprint(in.TargetPort))
	buf.WriteString("/")
	buf.WriteString(string(in.Protocol))

	return buf.String()
}

type PortDef struct {
	Hostname   string   `json:"hostname,omitempty"`
	Protocol   Protocol `json:"protocol,omitempty"`
	Publish    bool     `json:"publish,omitempty"`
	Port       int32    `json:"port,omitempty"`
	TargetPort int32    `json:"targetPort,omitempty"`
}

func (in PortDef) Complete() PortDef {
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

type File struct {
	Mode    string          `json:"mode,omitempty"`
	Content string          `json:"content,omitempty"`
	Secret  SecretReference `json:"secret,omitempty"`
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

type NameValue struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

type EnvVar struct {
	Name   string          `json:"name,omitempty"`
	Value  string          `json:"value,omitempty"`
	Secret SecretReference `json:"secret,omitempty"`
}

type SecretReference struct {
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

type ScopedLabel struct {
	ResourceType string `json:"resourceType,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
	Key          string `json:"key,omitempty"`
	Value        string `json:"value,omitempty"`
}

type PolicyRule struct {
	rbacv1.PolicyRule `json:",inline"`
	Scopes            []string `json:"scopes,omitempty"`
}

func (p PolicyRule) IsAccountScoped() bool {
	for _, scope := range p.Scopes {
		if scope == "" {
			return true
		}
		if scope == "cluster" {
			return true
		}
		if scope == "account" {
			return true
		}
	}
	return false
}

func (p PolicyRule) IsProjectScoped() bool {
	if len(p.Scopes) == 0 {
		return true
	}
	for _, scope := range p.Scopes {
		if scope == "project" {
			return true
		}
	}
	return false
}

func (p PolicyRule) ResolveNamespaces(currentNamespace string) (result []string) {
	namespaces := map[string]struct{}{}
	if p.IsProjectScoped() {
		namespaces[currentNamespace] = struct{}{}
	}
	if p.IsAccountScoped() {
		namespaces[""] = struct{}{}
	}
	for _, namespace := range p.Namespaces() {
		namespaces[namespace] = struct{}{}
	}

	for namespace := range namespaces {
		result = append(result, namespace)
	}

	sort.Strings(result)
	return
}

func (p PolicyRule) Namespaces() (result []string) {
	for _, scope := range p.Scopes {
		if strings.HasPrefix(scope, "namespace:") {
			result = append(result, strings.TrimPrefix(scope, "namespace:"))
		}
	}
	return
}

type Permissions struct {
	ServiceName string       `json:"serviceName,omitempty"`
	Rules       []PolicyRule `json:"rules,omitempty"`
	// Deprecated, use Rules with the 'scopes: ["cluster"]' field
	ZZ_ClusterRules []PolicyRule `json:"clusterRules,omitempty"`
}

func (in Permissions) GetRules() []PolicyRule {
	result := in.Rules
	for _, rule := range in.ZZ_ClusterRules {
		if len(rule.Scopes) == 0 {
			rule.Scopes = append(rule.Scopes, "cluster")
		}
		result = append(result, rule)
	}
	return result
}

func (in *Permissions) HasRules() bool {
	if in == nil {
		return false
	}
	return len(in.Rules) > 0 || len(in.ZZ_ClusterRules) > 0
}

func (in *Permissions) Get() Permissions {
	if in == nil {
		return Permissions{}
	}
	return *in
}

func FindPermission(serviceName string, perms []Permissions) Permissions {
	for _, perm := range perms {
		if serviceName == perm.ServiceName {
			return perm
		}
	}
	return Permissions{}
}

type Files map[string]File

type CommandSlice []string

type EnvVars []EnvVar

type NameValues []NameValue

type Probes []Probe

type Ports []PortDef

type Dependencies []Dependency

type ScopedLabels []ScopedLabel

type Container struct {
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	Dirs         map[string]VolumeMount `json:"dirs,omitempty"`
	Files        Files                  `json:"files,omitempty"`
	Image        string                 `json:"image,omitempty"`
	Build        *Build                 `json:"build,omitempty"`
	Command      CommandSlice           `json:"command,omitempty"`
	Interactive  bool                   `json:"interactive,omitempty"`
	Entrypoint   CommandSlice           `json:"entrypoint,omitempty"`
	Environment  EnvVars                `json:"environment,omitempty"`
	WorkingDir   string                 `json:"workingDir,omitempty"`
	Ports        Ports                  `json:"ports,omitempty"`
	Probes       Probes                 `json:"probes"` // Don't omitempty so that nil vs empty is recorded
	Dependencies Dependencies           `json:"dependencies,omitempty"`
	Permissions  *Permissions           `json:"permissions,omitempty"`
	ComputeClass *string                `json:"class,omitempty"`
	Memory       *int64                 `json:"memory,omitempty"`

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
	Image      string      `json:"image,omitempty"`
	Build      *Build      `json:"containerBuild,omitempty"`
	AcornBuild *AcornBuild `json:"build,omitempty"`
}

type AppSpec struct {
	Labels      map[string]string        `json:"labels,omitempty"`
	Annotations map[string]string        `json:"annotations,omitempty"`
	Containers  map[string]Container     `json:"containers,omitempty"`
	Jobs        map[string]Container     `json:"jobs,omitempty"`
	Images      map[string]Image         `json:"images,omitempty"`
	Volumes     map[string]VolumeRequest `json:"volumes,omitempty"`
	Secrets     map[string]Secret        `json:"secrets,omitempty"`
	Acorns      map[string]Acorn         `json:"acorns,omitempty"`
	Routers     map[string]Router        `json:"routers,omitempty"`
	Services    map[string]Service       `json:"services,omitempty"`
}

type Route struct {
	Path              string   `json:"path,omitempty"`
	TargetServiceName string   `json:"targetServiceName,omitempty"`
	TargetPort        int      `json:"targetPort,omitempty"`
	PathType          PathType `json:"pathType,omitempty"`
}

type Routes []Route

type Router struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Routes      Routes            `json:"routes,omitempty"`
}

type Acorn struct {
	Labels              ScopedLabels    `json:"labels,omitempty"`
	Annotations         ScopedLabels    `json:"annotations,omitempty"`
	Image               string          `json:"image,omitempty"`
	Build               *AcornBuild     `json:"build,omitempty"`
	Profiles            []string        `json:"profiles,omitempty"`
	DeployArgs          GenericMap      `json:"deployArgs,omitempty"`
	Publish             PortBindings    `json:"publish,omitempty"`
	Environment         NameValues      `json:"environment,omitempty"`
	Secrets             SecretBindings  `json:"secrets,omitempty"`
	Volumes             VolumeBindings  `json:"volumes,omitempty"`
	Links               ServiceBindings `json:"links,omitempty"`
	AutoUpgrade         *bool           `json:"autoUpgrade,omitempty"`
	NotifyUpgrade       *bool           `json:"notifyUpgrade,omitempty"`
	AutoUpgradeInterval string          `json:"autoUpgradeInterval,omitempty"`
	Memory              MemoryMap       `json:"memory,omitempty"`
	ComputeClasses      ComputeClassMap `json:"computeClasses,omitempty"`
}

type Secret struct {
	External    string            `json:"external,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Type        string            `json:"type,omitempty"`
	Params      GenericMap        `json:"params,omitempty"`
	Data        map[string]string `json:"data,omitempty"`
}

type AccessModes []AccessMode

type VolumeRequest struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Class       string            `json:"class,omitempty"`
	Size        Quantity          `json:"size,omitempty"`
	AccessModes AccessModes       `json:"accessModes,omitempty"`
}

// Workload to its memory
type MemoryMap map[string]*int64

// Workload to its class
type ComputeClassMap map[string]string

type GeneratedService struct {
	Job string `json:"job,omitempty"`
}

type Service struct {
	Labels              ScopedLabels      `json:"labels,omitempty"`
	Annotations         ScopedLabels      `json:"annotations,omitempty"`
	Default             bool              `json:"default,omitempty"`
	External            string            `json:"external,omitempty"`
	Address             string            `json:"address,omitempty"`
	Ports               Ports             `json:"ports,omitempty"`
	Container           string            `json:"container,omitempty"`
	Data                GenericMap        `json:"data,omitempty"`
	Generated           *GeneratedService `json:"generated,omitempty"`
	Image               string            `json:"image,omitempty"`
	Build               *AcornBuild       `json:"build,omitempty"`
	ServiceArgs         GenericMap        `json:"serviceArgs,omitempty"`
	Environment         NameValues        `json:"environment,omitempty"`
	Secrets             SecretBindings    `json:"secrets,omitempty"`
	Links               ServiceBindings   `json:"links,omitempty"`
	Permissions         []Permissions     `json:"permissions,omitempty"`
	AutoUpgrade         *bool             `json:"autoUpgrade,omitempty"`
	NotifyUpgrade       *bool             `json:"notifyUpgrade,omitempty"`
	AutoUpgradeInterval string            `json:"autoUpgradeInterval,omitempty"`
	Memory              MemoryMap         `json:"memory,omitempty"`
}

func (s Service) GetJob() string {
	if s.Generated == nil {
		return ""
	}
	return s.Generated.Job
}
