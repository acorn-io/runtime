package imageallowrules

import (
	"testing"
)

func TestMatchContext(t *testing.T) {
	testcases := []struct {
		name        string
		pattern     string
		image       string
		shouldMatch bool
	}{
		{
			// this is expected, since we're catching empty patterns earlier
			name:        "empty pattern",
			pattern:     "",
			image:       "docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "empty image",
			pattern:     "docker.io/library/alpine:latest",
			image:       "",
			shouldMatch: false,
		},
		{
			name:        "exact match",
			pattern:     "docker.io/library/alpine:latest",
			image:       "docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		// shell filename pattern matching with ? and * only
		{
			name:        "globbing match",
			pattern:     "docker.io/**/*",
			image:       "docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "globbing mismatch",
			pattern:     "docker.io/library/*",
			image:       "docker.io/foo/alpine:latest",
			shouldMatch: false,
		},
		{
			name:        "prefix match",
			pattern:     "docker.io/",
			image:       "docker.io/library/alpine:latest",
			shouldMatch: true,
		},
		{
			name:        "prefix mismatch",
			pattern:     "docker.io/library",
			image:       "docker.io/foo/alpine:latest",
			shouldMatch: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := matchContext(tc.pattern, tc.image)
			if (err != nil && tc.shouldMatch) || (err == nil && !tc.shouldMatch) {
				if tc.shouldMatch {
					t.Errorf("expected pattern %s to match %s, but it didn't", tc.pattern, tc.image)
				} else {
					t.Errorf("expected pattern %s to not match %s, but it did", tc.pattern, tc.image)
				}
			}
		})
	}
}
