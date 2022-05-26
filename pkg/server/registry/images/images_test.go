package images

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/stretchr/testify/assert"

	"testing"
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
}
