package v1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

type Build struct {
	Context    string `json:"context,omitempty"`
	Dockerfile string `json:"dockerfile,omitempty"`
	Target     string `json:"target,omitempty"`
}

// Hash will return the same hash for the same object. There's no guarentee
// overtime that the same hash will be produced.  This is intended for multiple
// execution of the same compilation of herd to produce that same value to provide
// a small about of caching. To not rely on this hash to not change over time.
func (in Build) Hash() string {
	hash := sha256.New()
	err := json.NewEncoder(hash).Encode(in)
	if err != nil {
		// this should never happen
		panic("failed to marshall struct")
	}
	result := hash.Sum(nil)
	return hex.EncodeToString(result[:])
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
