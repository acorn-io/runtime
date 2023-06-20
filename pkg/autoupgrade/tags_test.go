package autoupgrade

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
)

func TestTags(t *testing.T) {
	// Simplest numeric
	test(t, "#", 2, []string{"10", "12", "20", "v21", "30.1"})

	// Simplest alphabetical
	test(t, "*", 2, []string{"alpha", "beta", "zeta", "xtra"})

	// Simple "I can't believe it's not semver" usecases
	test(t, "v#.#", 1, []string{"v1.0", "v1.2", "v0.9", "v1.1"})
	test(t, "v#.#", 3, []string{"v1.0", "1.2", "v0.9", "v1.1"})
	test(t, "v#.#", 0, []string{"v1.0", "1.2", "v0.9", "v1"})
	test(t, "v#.#", 2, []string{"v1.0-rc1", "1.2", "v0.9", "v1"})

	// sort pre-release part alphabetically
	test(t, "v#.#-*", 1, []string{"v1.0-rc10", "v1.0-rc2", "v1.0-alpha1", "v1.0-beta"})

	// RCs only and sort numerically
	test(t, "v#.#-rc.#", 0, []string{"v1.0-rc.10", "v1.0-rc.2", "v1.0-alpha.100", "v1.0-rc11"})

	// I don't care about the "pre-release" part, just give me the latest 1.x
	test(t, "v#.#-**", 1, []string{"v1.0-a", "v1.1-b", "v1.0-c", "v1.0-z"})

	// I don't care about the anything other than the numerical version, just give me the latest 1.x
	test(t, "v#.#**", 2, []string{"v1.0-a", "v1.1-b", "v1.2", "v1.0-c", "v1.0-z"})

	// an alpha sort segment, followed by a dot, followed by a numeric sort segment
	test(t, "v#.#-*.#", 2, []string{"v1.1-a.9", "v1.1-b.1", "v1.1-b.2", "v1.1-b.0"})

	// Weird, but does what it is supposed to, which is to just do an alpha sort on the tag
	test(t, "*", 2, []string{"v1.0-alpha.100", "v1.0-beta", "v1.0-zeta"})
}

func test(t *testing.T, pattern string, expectedIndex int, tags []string) {
	t.Helper()
	latest, err := FindLatest("", pattern, tags)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, tags[expectedIndex], latest)
}

func TestGetTagsForImagePattern(t *testing.T) {
	testCases := []struct {
		name             string
		currentImage     string
		remoteTags       []string
		localTags        []string
		expectRemote     bool
		expectedRegistry string
	}{
		{
			name:             "remote, no specified registry",
			currentImage:     "myrepo/myimage:v1.0.0",
			remoteTags:       []string{"v1.0.0", "v1.0.1"},
			localTags:        []string{"v2.0.0"},
			expectRemote:     true,
			expectedRegistry: "index.docker.io",
		},
		{
			name:             "remote, docker.io",
			currentImage:     "docker.io/myrepo/myimage:v1.0.0",
			remoteTags:       []string{"v1.0.0", "v1.0.1"},
			localTags:        []string{"v2.0.0"},
			expectRemote:     true,
			expectedRegistry: "index.docker.io",
		},
		{
			name:             "remote, index.docker.io",
			currentImage:     "index.docker.io/myrepo/myimage:v1.0.1",
			remoteTags:       []string{"v1.0.0", "v1.0.1"},
			localTags:        []string{"v2.0.0"},
			expectRemote:     true,
			expectedRegistry: "index.docker.io",
		},
		{
			name:             "local, no specified registry",
			currentImage:     "myrepo/myimage:v1.0.0",
			remoteTags:       []string{},
			localTags:        []string{"v2.0.0"},
			expectRemote:     false,
			expectedRegistry: defaultNoReg,
		},
		{
			name:             "remote, ghcr.io",
			currentImage:     "ghcr.io/myrepo/myimage:v1.0.1",
			remoteTags:       []string{"v1.0.0", "v1.1.0"},
			localTags:        []string{"v2.0.0"},
			expectRemote:     true,
			expectedRegistry: "ghcr.io",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			client := testDaemonClient{
				localTags:  tt.localTags,
				remoteTags: tt.remoteTags,
			}

			currentRef, foundTags, err := getTagsForImagePattern(context.Background(), client, "my-namespace", tt.currentImage)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedRegistry, currentRef.Context().RegistryStr())
			if tt.expectRemote {
				assert.Equal(t, tt.remoteTags, foundTags)
			} else {
				assert.Equal(t, tt.localTags, foundTags)
			}
		})
	}
}

type testDaemonClient struct {
	localTags, remoteTags []string
}

func (c testDaemonClient) listTags(_ context.Context, _, _ string, _ ...remote.Option) ([]string, error) {
	return c.remoteTags, nil
}

func (c testDaemonClient) getTagsMatchingRepo(_ context.Context, _ name.Reference, _, _ string) ([]string, error) {
	return c.localTags, nil
}

// We don't care about these functions

func (c testDaemonClient) getConfig(_ context.Context) (*apiv1.Config, error) {
	return nil, nil
}

func (c testDaemonClient) listAppInstances(_ context.Context) ([]v1.AppInstance, error) {
	return nil, nil
}

func (c testDaemonClient) updateAppStatus(_ context.Context, _ *v1.AppInstance) error {
	return nil
}

func (c testDaemonClient) imageDigest(_ context.Context, _, _ string, _ ...remote.Option) (string, error) {
	return "", nil
}

func (c testDaemonClient) resolveLocalTag(_ context.Context, _, _ string) (string, bool, error) {
	return "", false, nil
}

func (c testDaemonClient) checkImageAllowed(_ context.Context, _, _ string) error {
	return nil
}
