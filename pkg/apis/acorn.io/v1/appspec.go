package v1

const (
	VolumeRequestTypeEphemeral = "ephemeral"

	AccessModeReadWriteMany AccessMode = "readWriteMany"
	AccessModeReadWriteOnce AccessMode = "readWriteOnce"
	AccessModeReadOnlyMany  AccessMode = "readOnlyMany"
)

type AccessMode string

type AcornBuild struct {
	OriginalImage string     `json:"originalImage,omitempty"`
	Context       string     `json:"context,omitempty"`
	Acornfile     string     `json:"acornfile,omitempty"`
	Params        GenericMap `json:"params,omitempty"`
}

type Build struct {
	Context     string            `json:"context,omitempty"`
	Dockerfile  string            `json:"dockerfile,omitempty"`
	Target      string            `json:"target,omitempty"`
	BaseImage   string            `json:"baseImage,omitempty"`
	ContextDirs map[string]string `json:"contextDirs,omitempty"`
	Args        map[string]string `json:"args,omitempty"`
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
	ProtocolTCP   = Protocol("tcp")
	ProtocolUDP   = Protocol("udp")
	ProtocolHTTP  = Protocol("http")
	ProtocolHTTPS = Protocol("https")
)

type PublishProtocol string

var (
	PublishProtocolTCP  = PublishProtocol("tcp")
	PublishProtocolUDP  = PublishProtocol("udp")
	PublishProtocolHTTP = PublishProtocol("http")
)

type AppPort struct {
	Port       int32    `json:"port,omitempty" wrangler:"required"`
	TargetPort int32    `json:"targetPort,omitempty" wrangler:"required"`
	Protocol   Protocol `json:"protocol,omitempty" wrangler:"required"`
	Publish    bool     `json:"publish,omitempty"`
}

type Port struct {
	Port          int32    `json:"port,omitempty"`
	ContainerPort int32    `json:"containerPort,omitempty"`
	Protocol      Protocol `json:"protocol,omitempty"`
	Publish       bool     `json:"publish,omitempty"`
}

type FileSecret struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}

type File struct {
	Content string     `json:"content,omitempty"`
	Secret  FileSecret `json:"secret,omitempty"`
}

type VolumeSecretMount struct {
	Name string `json:"name,omitempty"`
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
	Name     string `json:"name,omitempty"`
	Key      string `json:"key,omitempty"`
	Optional *bool  `json:"optional,omitempty"`
}

type Alias struct {
	Name string `json:"name,omitempty"`
}

type Container struct {
	Dirs        map[string]VolumeMount `json:"dirs,omitempty"`
	Files       map[string]File        `json:"files,omitempty"`
	Image       string                 `json:"image,omitempty"`
	Build       *Build                 `json:"build,omitempty"`
	Command     []string               `json:"command,omitempty"`
	Interactive bool                   `json:"interactive,omitempty"`
	Entrypoint  []string               `json:"entrypoint,omitempty"`
	Environment []EnvVar               `json:"environment,omitempty"`
	WorkingDir  string                 `json:"workingDir,omitempty"`
	Ports       []Port                 `json:"ports,omitempty"`

	// Scale is only available on containers, not sidecars or jobs
	Scale *int32 `json:"scale,omitempty"`

	// Alias is only available on containers, not sidecars or jobs
	Aliases []Alias `json:"aliases,omitempty"`

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
	Image   string          `json:"image,omitempty"`
	Build   *AcornBuild     `json:"build,omitempty"`
	Params  GenericMap      `json:"params,omitempty"`
	Ports   []AppPort       `json:"ports,omitempty"`
	Secrets []SecretBinding `json:"secrets,omitempty"`
	Volumes []VolumeBinding `json:"volumes,omitempty"`
}

type Secret struct {
	Type     string            `json:"type,omitempty"`
	Params   GenericMap        `json:"params,omitempty"`
	Optional *bool             `json:"optional,omitempty"`
	Data     map[string]string `json:"data,omitempty"`
}

type VolumeRequest struct {
	Class       string       `json:"class,omitempty"`
	Size        int64        `json:"size,omitempty"`
	ContextPath string       `json:"contextPath,omitempty"`
	AccessModes []AccessMode `json:"accessModes,omitempty"`
}
