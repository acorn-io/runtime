package v1

type Build struct {
	Context    string `json:"context,omitempty"`
	Dockerfile string `json:"dockerfile,omitempty"`
}

type Container struct {
	Image string `json:"image,omitempty"`
	Build Build  `json:"build,omitempty"`
}

type AppSpec struct {
	Containers map[string]Container `json:"containers,omitempty"`
}
