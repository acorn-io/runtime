package v1

type ContainerImageBuildSpec struct {
	Image    string                    `json:"image,omitempty"`
	Build    *Build                    `json:"build,omitempty"`
	Sidecars map[string]ImageBuildSpec `json:"sidecars,omitempty"`
}

type ImageBuildSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type BuildSpec struct {
	Containers map[string]ContainerImageBuildSpec `json:"containers,omitempty"`
	Images     map[string]ImageBuildSpec          `json:"images,omitempty"`
}
