package client

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestMergeEnv(t *testing.T) {
	existing := []v1.NameValue{
		{
			Name:  "a",
			Value: "b",
		},
		{
			Name:  "c",
			Value: "d",
		},
	}
	newValues := mergeEnv(existing, []v1.NameValue{
		{
			Name:  "e",
			Value: "f",
		},
		{
			Name:  "c",
			Value: "updated",
		},
	})
	assert.Equal(t, "a", newValues[0].Name)
	assert.Equal(t, "b", newValues[0].Value)
	assert.Equal(t, "c", newValues[1].Name)
	assert.Equal(t, "updated", newValues[1].Value)
	assert.Equal(t, "e", newValues[2].Name)
	assert.Equal(t, "f", newValues[2].Value)
}
