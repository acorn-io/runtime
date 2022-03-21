package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	VolumeRequestTypeEphemeral = "ephemeral"

	AccessModeReadWriteMany    AccessMode = "readWriteMany"
	AccessModeReadWriteOnce    AccessMode = "readWriteOnce"
	AccessModeReadOnlyMany     AccessMode = "readOnlyMany"
	AccessModeReadWriteOncePod AccessMode = "readWriteOncePod"
)

type AccessMode string

type Build struct {
	Context     string            `json:"context,omitempty"`
	Dockerfile  string            `json:"dockerfile,omitempty"`
	Target      string            `json:"target,omitempty"`
	BaseImage   string            `json:"baseImage,omitempty"`
	ContextDirs map[string]string `json:"contextDirs,omitempty"`
}

func (in Build) BaseBuild() Build {
	return Build{
		Context:    in.Context,
		Dockerfile: in.Dockerfile,
		Target:     in.Target,
	}
}

// Hash will return the same hash for the same object. There's no guarentee
// overtime that the same hash will be produced.  This is intended for multiple
// execution of the same compilation of herd to produce that same value to provide
// a small about of caching. To not rely on this hash to not change over time.
func (in Build) Hash() string {
	hash := sha256.New()
	err := json.NewEncoder(hash).Encode(in)
	if err != nil {
		// this should never happen
		panic("failed to marshall struct")
	}
	result := hash.Sum(nil)
	return hex.EncodeToString(result[:])
}

type Protocol string

var (
	ProtocolTCP   = Protocol("tcp")
	ProtocolUDP   = Protocol("udp")
	ProtocolHTTP  = Protocol("http")
	ProtocolHTTPS = Protocol("https")
)

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
	Content string      `json:"content,omitempty"`
	Secret  *FileSecret `json:"secret,omitempty"`
}

type VolumeSecretMount struct {
	Name string `json:"name,omitempty"`
}

type VolumeMount struct {
	Volume     string             `json:"volume,omitempty"`
	SubPath    string             `json:"subPath,omitempty"`
	ContextDir string             `json:"contextDir,omitempty"`
	Secret     *VolumeSecretMount `json:"secret,omitempty"`
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
	Images     map[string]Image         `json:"images,omitempty"`
	Volumes    map[string]VolumeRequest `json:"volumes,omitempty"`
}

type VolumeRequest struct {
	Class       string       `json:"class,omitempty"`
	Size        int64        `json:"size,omitempty"`
	ContextPath string       `json:"contextPath,omitempty"`
	AccessModes []AccessMode `json:"accessModes,omitempty"`
}
