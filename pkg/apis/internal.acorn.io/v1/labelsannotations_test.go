package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScopedLabels(t *testing.T) {
	simpleTest(t, "k=v", ScopedLabel{Key: "k", Value: "v"})

	// all the supported resource types, singular and plural
	for _, rt := range []string{"container", "job", "volume", "secret", "acorn", "metadata", "router", "service"} {
		// with the resourceType specified. Example: --label container:k=v
		simpleTest(t, rt+":k=v", ScopedLabel{ResourceType: rt, Key: "k", Value: "v"})
		simpleTest(t, rt+"s:k=v", ScopedLabel{ResourceType: rt, Key: "k", Value: "v"})
		// with the resourceType and resourceName specified. Example: --label container:n:k=v
		simpleTest(t, rt+":n:k=v", ScopedLabel{ResourceType: rt, ResourceName: "n", Key: "k", Value: "v"})
		simpleTest(t, rt+"s:n:k=v", ScopedLabel{ResourceType: rt, ResourceName: "n", Key: "k", Value: "v"})
	}

	_, err := ParseScopedLabels("a:b:c:k=v")
	assert.Error(t, err)
	_, err = ParseScopedLabels("something:n:k=v")
	assert.Error(t, err)
	_, err = ParseScopedLabels("container::k=v")
	assert.Error(t, err)
}

func simpleTest(t *testing.T, input string, expected ScopedLabel) {
	t.Helper()

	actual, err := ParseScopedLabels(input)
	assert.NoError(t, err)
	assert.Len(t, actual, 1)
	assert.Equal(t, expected, actual[0])
}
