package v1

type ContainerImageBuilderSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
	// Sidecars is only populated for non-sidecar containers
	Sidecars map[string]ContainerImageBuilderSpec `json:"sidecars,omitempty"`
}

type ImageBuilderSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type AcornBuilderSpec struct {
	Image string      `json:"image,omitempty"`
	Build *AcornBuild `json:"build,omitempty"`
}

type BuilderSpec struct {
	Platforms  []Platform                           `json:"platforms,omitempty"`
	Containers map[string]ContainerImageBuilderSpec `json:"containers,omitempty"`
	Jobs       map[string]ContainerImageBuilderSpec `json:"jobs,omitempty"`
	Images     map[string]ImageBuilderSpec          `json:"images,omitempty"`
	Acorns     map[string]AcornBuilderSpec          `json:"acorns,omitempty"`
}

type ParamSpec struct {
	Params []Param `json:"params,omitempty"`
}

type Param struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema,omitempty"`
}
