package v1

type ContainerData struct {
	Image    string               `json:"image,omitempty"`
	Sidecars map[string]ImageData `json:"sidecars,omitempty"`
}

type ImageData struct {
	Image string `json:"image,omitempty"`
}

type ImagesData struct {
	Containers map[string]ContainerData `json:"containers,omitempty"`
	Jobs       map[string]ContainerData `json:"jobs,omitempty"`
	Images     map[string]ImageData     `json:"images,omitempty"`
	Acorns     map[string]ImageData     `json:"acorns,omitempty"`
}
