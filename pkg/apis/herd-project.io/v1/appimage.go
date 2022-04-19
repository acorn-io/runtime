package v1

type AppImage struct {
	ID          string     `json:"id,omitempty"`
	Herdfile    string     `json:"herdfile,omitempty"`
	ImageData   ImagesData `json:"imageData,omitempty"`
	BuildParams GenericMap `json:"buildParams,omitempty"`
}

type Platform struct {
	Architecture string   `json:"architecture"`
	OS           string   `json:"os"`
	OSVersion    string   `json:"os.version,omitempty"`
	OSFeatures   []string `json:"os.features,omitempty"`
	Variant      string   `json:"variant,omitempty"`
}
