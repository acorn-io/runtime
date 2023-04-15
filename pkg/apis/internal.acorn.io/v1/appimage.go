package v1

type AppImage struct {
	// ID is the "image ID" that Name resolves to, which might be the same as Name or a string matching
	// ImageInstance.Name
	ID string `json:"id,omitempty"`
	// Name is the image name requested by the user of any format
	Name      string     `json:"name,omitempty"`
	Digest    string     `json:"digest,omitempty"`
	Acornfile string     `json:"acornfile,omitempty"`
	ImageData ImagesData `json:"imageData,omitempty"`
	BuildArgs GenericMap `json:"buildArgs,omitempty"`
	VCS       VCS        `json:"vcs,omitempty"`
}

type VCS struct {
	Revision string `json:"revision,omitempty"`
	Modified bool   `json:"modified,omitempty"`
}

type Platform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}
