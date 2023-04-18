package helper

// TestImage used for unmarshalling json output from acorn image <image-name> -o=json
type TestImage struct {
	Name       string `json:"name"`
	Digest     string `json:"digest"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
}

// TestProject used for unmarshalling json output from acorn project <project-name> -o=json
type TestProject struct {
	Name    string   `json:"name"`
	Regions []string `json:"regions"`
	Default bool     `json:"default"`
}
