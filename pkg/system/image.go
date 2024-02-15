package system

import (
	"fmt"
	"os"

	"github.com/acorn-io/runtime/pkg/version"
)

var (
	InstallImage     = "images.acornlabs.com/acorn-io/runtime"
	LocalImage       = "acorn-local"
	LocalDockerImage = os.Getenv("ACORN_DOCKER_IMAGE")
	LocalImageBind   = "ghcr.io/acorn-io/acorn-local-bind:latest"
	LocalNode        = "acorn-node"
	DefaultBranch    = "main"
	devTag           = "v0.0.0-dev"
)

func IsLocal() bool {
	return DefaultImage() == LocalImage
}

func DefaultImage() string {
	img := os.Getenv("ACORN_IMAGE")
	if img != "" {
		return img
	}
	var image = fmt.Sprintf("%s:%s", InstallImage, version.Tag)
	if version.Tag == devTag {
		image = fmt.Sprintf("%s:%s", InstallImage, DefaultBranch)
	}
	return image
}
