package system

import (
	"fmt"
	"os"

	"github.com/acorn-io/acorn/pkg/version"
)

var (
	InstallImage  = "ghcr.io/acorn-io/acorn"
	DefaultBranch = "main"
	devTag        = "v0.0.0-dev"
)

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
