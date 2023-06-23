package client

import (
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
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

func TestLabelsAndAnnotations(t *testing.T) {
	opts := &AppRunOptions{
		Labels: []v1.ScopedLabel{
			{
				ResourceType: "metadata",
				ResourceName: "",
				Key:          "label1",
				Value:        "v1",
			},
			{
				ResourceType: "",
				ResourceName: "",
				Key:          "label2",
				Value:        "v2",
			},
			{
				ResourceType: "container",
				ResourceName: "",
				Key:          "label3",
				Value:        "v3",
			},
		},
		Annotations: []v1.ScopedLabel{
			{
				ResourceType: "metadata",
				ResourceName: "",
				Key:          "anno1",
				Value:        "v1",
			},
			{
				ResourceType: "",
				ResourceName: "",
				Key:          "anno2",
				Value:        "v2",
			},
			{
				ResourceType: "container",
				ResourceName: "",
				Key:          "anno3",
				Value:        "v3",
			},
		},
	}
	app := ToApp("ns", "image1", opts)
	assert.Equal(t, "v1", app.Labels["label1"])
	assert.Equal(t, "v2", app.Labels["label2"])
	assert.NotContains(t, app.Labels, "label3")

	assert.Equal(t, "v1", app.Annotations["anno1"])
	assert.Equal(t, "v2", app.Annotations["anno2"])
	assert.NotContains(t, app.Annotations, "anno3")
}
