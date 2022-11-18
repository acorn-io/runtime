package autoupgrade

import (
	"testing"

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
