package v1

type ImageBuildSpec struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type BuildSpec struct {
	Containers map[string]ImageBuildSpec `json:"containers,omitempty"`
	Images     map[string]ImageBuildSpec `json:"images,omitempty"`
}
