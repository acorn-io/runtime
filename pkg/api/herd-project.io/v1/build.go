package v1

type ContainerBuildSpec struct {
	Image string `json:"image,omitempty"`
	Build Build  `json:"build,omitempty"`
}

type BuildSpec struct {
	Containers map[string]ContainerBuildSpec `json:"containers,omitempty"`
}
