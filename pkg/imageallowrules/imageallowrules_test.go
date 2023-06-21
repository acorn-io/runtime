package imageallowrules

import (
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestImageCovered(t *testing.T) {
	testcases := []struct {
		name        string
		pattern     string
		image       string
		shouldMatch bool
	}{
		{
			name:        "empty pattern",
			pattern:     "",
			image:       "index.docker.io/library/alpine:latest",
			shouldMatch: false,
		},
		{
			name:        "exact match",
			pattern:     "index.docker.io/library/alpine:latest",
			image:       "index.docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "registry match",
			pattern:     "index.docker.io/**",
			image:       "index.docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "repo and tag alpha match",
			pattern:     "index.docker.io/library/*:*",
			image:       "index.docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "tag semver match",
			pattern:     "index.docker.io/library/alpine:v#.#.#",
			image:       "index.docker.io/library/alpine:v1.0.1",
			shouldMatch: true,
		},
		{
			name:        "repo alpha and tag semver",
			pattern:     "index.docker.io/library/*:v#.#.#",
			image:       "index.docker.io/library/alpine:v1.0.1",
			shouldMatch: true,
		},
		{
			name:        "repo path wildcard with specific repo and tag semver",
			pattern:     "index.docker.io/**/alpine:v#.#.#",
			image:       "index.docker.io/library/foo/alpine:v1.0.1",
			shouldMatch: true,
		},
		{
			name:        "repo path single element wildcard with specific repo and tag semver",
			pattern:     "index.docker.io/*/alpine:v#.#.#",
			image:       "index.docker.io/library/alpine:v1.0.1",
			shouldMatch: true,
		},
		{
			name:        "mismatch wrong repo: repo path single element wildcard with specific repo and tag semver",
			pattern:     "index.docker.io/*/alpine:v#.#.#",
			image:       "index.docker.io/library/notalpine:v1.0.1",
			shouldMatch: false,
		},
		{
			name:        "mismatch subrepos: repo path single element wildcard with specific repo and tag semver",
			pattern:     "index.docker.io/*/alpine:v#.#.#",
			image:       "index.docker.io/library/foo/alpine:v1.0.1",
			shouldMatch: false,
		},
		{
			name:        "match by full ID",
			pattern:     "e67e444786a869161b26fa00f4993bbdeba3da677043e0bead8747d7a05eb150",
			image:       "e67e444786a869161b26fa00f4993bbdeba3da677043e0bead8747d7a05eb150",
			shouldMatch: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := name.ParseReference(tc.image, name.WithDefaultRegistry(""), name.WithDefaultTag(""))
			if err != nil {
				t.Fatalf("failed to parse image %s: %v", tc.image, err)
			}

			match := imageCovered(ref, "", v1.ImageAllowRuleInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
				Images: []string{tc.pattern},
			})

			assert.Equal(t, tc.shouldMatch, match)
		})
	}
}
