package images

import (
	"testing"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestFindMatchingImage(t *testing.T) {
	images := []apiv1.Image{
		{
			Digest: "sha256:12345678",
		},
		{
			Digest: "sha256:987654321",
			Tags: []string{
				"foo:latest",
			},
		},
		{
			Digest: "sha256:123409876",
			Tags: []string{
				"foo/bar",
				"foo/bar:dev",
			},
		},
		{
			Digest: "sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb",
			Tags: []string{
				"foo/bar",
				"spam/eggs:v1",
			},
		},
	}
	il := apiv1.ImageList{
		Items: images,
	}

	// pass: digest prefix
	image, ref, err := findImageMatch(il, "12345")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sha256:12345678", image.Digest)
	assert.Equal(t, "", ref)

	// err: ambiguous digest prefix
	_, _, err = findImageMatch(il, "123")
	assert.Error(t, err)
	assert.Equal(t, "image identifier not unique: 123", err.Error())

	// pass: full digest reference
	image, ref, err = findImageMatch(il, "foo/bar@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb", image.Digest)
	assert.Equal(t, "foo/bar", ref)

	// err: not found by full digest reference
	_, _, err = findImageMatch(il, "ghcr.io/acorn-io/library/hello-world@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb")
	assert.Error(t, err)
	assert.Equal(t, "images.api.acorn.io \"ghcr.io/acorn-io/library/hello-world@sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb\" not found", err.Error())

	// err: ambiguous reg/repo reference
	_, _, err = findImageMatch(il, "foo/bar")
	assert.Error(t, err)
	assert.Equal(t, "image identifier not unique: foo/bar", err.Error())

	// pass: full tag reference
	image, ref, err = findImageMatch(il, "spam/eggs:v1")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sha256:1a6c64d2ccd0bb035f9c8196d3bfe72a7fdbddc4530dfcb3ab2a0ab8afb57eeb", image.Digest)
	assert.Equal(t, "spam/eggs:v1", ref)

	// pass: repo without tag, defaulting to :latest
	image, ref, err = findImageMatch(il, "foo")
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sha256:987654321", image.Digest)
	assert.Equal(t, "foo:latest", ref)

	// err: tiny string that's neither a digest nor an existing tag
	// (it is a valid image tag, but not an acorn image tag item)
	_, _, err = findImageMatch(il, "dev")
	assert.Error(t, err)
	assert.Equal(t, "images.api.acorn.io \"dev\" not found", err.Error())

	// err: same as above, but with repo part
	_, _, err = findImageMatch(il, "bar:dev")
	assert.Error(t, err)
	assert.Equal(t, "images.api.acorn.io \"bar:dev\" not found", err.Error())
}
