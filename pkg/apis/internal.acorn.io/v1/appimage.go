package v1

type AppImage struct {
	// ID is the "image ID" that Name resolves to, which might be the same as Name or a string matching
	// ImageInstance.Name
	ID string `json:"id,omitempty"`
	// Name is the image name requested by the user of any format
	Name      string      `json:"name,omitempty"`
	Digest    string      `json:"digest,omitempty"`
	Acornfile string      `json:"acornfile,omitempty"`
	ImageData ImagesData  `json:"imageData,omitempty"`
	BuildArgs *GenericMap `json:"buildArgs,omitempty"`
	Profiles  []string    `json:"profiles,omitempty"`
	VCS       VCS         `json:"vcs,omitempty"`

	AcornfileV1 bool `json:"acornfileV1,omitempty"`
}

type VCS struct {
	Remotes  []string `json:"remotes,omitempty"`
	Revision string   `json:"revision,omitempty"`
	// Clean a true value indicates the build contained no modified or untracked files according to git
	Clean bool `json:"clean,omitempty"`
	// Modified a true value indicates the build contained modified files according to git
	Modified bool `json:"modified,omitempty"`
	// Untracked a true value indicates the build contained untracked files according to git
	Untracked bool `json:"untracked,omitempty"`
}

type Platform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}
