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

type BuilderSpec struct {
	Containers map[string]ContainerImageBuilderSpec `json:"containers,omitempty"`
	Images     map[string]ImageBuilderSpec          `json:"images,omitempty"`
}
