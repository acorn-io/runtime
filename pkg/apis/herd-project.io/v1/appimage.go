package v1

type AppImage struct {
	ID        string     `json:"id,omitempty"`
	Herdfile  string     `json:"herdfile,omitempty"`
	ImageData ImagesData `json:"imageData,omitempty"`
}
