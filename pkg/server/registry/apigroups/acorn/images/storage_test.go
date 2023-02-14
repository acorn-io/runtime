package images

import (
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestFindMatchingImage(t *testing.T) {
	images := []apiv1.Image{
		{
			Digest: "sha256:12345678",
		},
		{
			Digest: "sha256:987654321",
		},
		{
			Digest: "sha256:123409876",
		},
	}
	il := apiv1.ImageList{
		Items: images,
	}

	image, ref, err := findImageMatch(il, "12345")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sha256:12345678", image.Digest)
	assert.Equal(t, "", ref)

	_, _, err = findImageMatch(il, "123")
	assert.Error(t, err)

	_, _, err = findImageMatch(il, "ghcr.io/acorn-io/library/hello-world@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb")
	assert.Error(t, err)
	assert.Equal(t, "images.api.acorn.io \"ghcr.io/acorn-io/library/hello-world@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb\" not found", err.Error())
}
