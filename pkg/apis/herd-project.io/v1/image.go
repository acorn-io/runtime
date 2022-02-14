package v1

type ContainerData struct {
	Image string `json:"image,omitempty"`
}

type ImageData struct {
	Containers map[string]ContainerData `json:"containers,omitempty"`
	Images     map[string]ContainerData `json:"images,omitempty"`
}
