package config

import (
	"context"
	"testing"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestAcornDNSDisabledNoLookupsHappen(t *testing.T) {
	s := "not exactly disabled, but any string that doesn't equal" +
		" auto or enabled should be treated as disabled"
	_ = complete(context.Background(), &apiv1.Config{
		AcornDNS: &s,
	}, nil)
	// if a lookup is going to happen this method would panic as the getter is nil
}

func TestMergeConfigWithZeroLengthStringsArrayShouldNotOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{},
		AllowUserLabels:             []string{},
		PropagateProjectLabels:      []string{},
		PropagateProjectAnnotations: []string{},
		ClusterDomains:              []string{},
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"foo"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"foo"}, result.AllowUserLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{".acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithActualValueStringsArrayShouldOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"bar"},
		AllowUserLabels:             []string{"bar"},
		PropagateProjectLabels:      []string{"bar"},
		PropagateProjectAnnotations: []string{"bar", "brah"},
		ClusterDomains:              []string{"bar.acorn.io"},
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"bar"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"bar"}, result.AllowUserLabels)
	assert.Equal(t, []string{"bar"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{"bar", "brah"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{".bar.acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithNilStringsArrayShouldNotOverride(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        nil,
		AllowUserLabels:             nil,
		PropagateProjectLabels:      nil,
		PropagateProjectAnnotations: nil,
		ClusterDomains:              nil,
	}

	result := merge(oldConfig, newConfig)
	assert.Equal(t, []string{"foo"}, result.AllowUserAnnotations)
	assert.Equal(t, []string{"foo"}, result.AllowUserLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectLabels)
	assert.Equal(t, []string{"foo"}, result.PropagateProjectAnnotations)
	assert.Equal(t, []string{".acorn.io"}, result.ClusterDomains)
}

func TestMergeConfigWithEmptyStringStringsArrayShouldOverrideToNil(t *testing.T) {
	oldConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{"foo"},
		AllowUserLabels:             []string{"foo"},
		PropagateProjectLabels:      []string{"foo"},
		PropagateProjectAnnotations: []string{"foo"},
		ClusterDomains:              []string{".acorn.io"},
	}

	newConfig := &apiv1.Config{
		AllowUserAnnotations:        []string{""},
		AllowUserLabels:             []string{""},
		PropagateProjectLabels:      []string{""},
		PropagateProjectAnnotations: []string{""},
		ClusterDomains:              []string{""},
	}

	result := merge(oldConfig, newConfig)
	assert.Nil(t, result.AllowUserAnnotations)
	assert.Nil(t, result.AllowUserLabels)
	assert.Nil(t, result.PropagateProjectLabels)
	assert.Nil(t, result.PropagateProjectAnnotations)
	assert.Nil(t, result.ClusterDomains)
}
