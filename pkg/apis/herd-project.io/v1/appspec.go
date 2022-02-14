package v1

type Build struct {
	Context    string `json:"context,omitempty"`
	Dockerfile string `json:"dockerfile,omitempty"`
	Target     string `json:"target,omitempty"`
}

type Container struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type Image struct {
	Image string `json:"image,omitempty"`
	Build *Build `json:"build,omitempty"`
}

type AppSpec struct {
	Containers map[string]Container `json:"containers,omitempty"`
	Images     map[string]Image     `json:"images,omitempty"`
}
