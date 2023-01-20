package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func int64Ptr(i int64) *int64 {
	return &i
}

func TestMemoryToRequirements(t *testing.T) {
	const Mi = int64(1048576)
	type want struct {
		request int64
		limit   int64
	}
	tests := []struct {
		name           string
		specMemory     Memory
		container      Container
		containerName  string
		specMemDefault *int64
		specMemMaximum *int64
		want           want
		err            error
	}{
		{
			name:           "successful with default",
			specMemory:     Memory{},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(100 * Mi),
			specMemMaximum: int64Ptr(0),
			want:           want{request: 100 * Mi, limit: 100 * Mi},
			err:            nil,
		},
		{
			name:           "successful with setting from user",
			specMemory:     Memory{"onecontainer": int64Ptr(512 * Mi)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(0),
			specMemMaximum: int64Ptr(0),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			name:           "successful with setting from Acornfile",
			specMemory:     Memory{},
			container:      Container{Memory: int64Ptr(512 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(0),
			specMemMaximum: int64Ptr(0),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with user setting of 0 and max not being 0",
			specMemory:     Memory{"onecontainer": int64Ptr(0)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(512 * Mi),
			specMemMaximum: int64Ptr(512 * Mi),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with Acornfile setting of 0 and max not being 0",
			specMemory:     Memory{},
			container:      Container{Memory: int64Ptr(0)},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(512 * Mi),
			specMemMaximum: int64Ptr(512 * Mi),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with default setting of 0 and max not being 0",
			specMemory:     Memory{},
			container:      Container{Memory: int64Ptr(0)},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(0),
			specMemMaximum: int64Ptr(512 * Mi),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			name:           "successful overwrite of Acornfile with user setting",
			specMemory:     Memory{"onecontainer": int64Ptr(512 * Mi)},
			container:      Container{Memory: int64Ptr(256 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(0),
			specMemMaximum: int64Ptr(0),
			want:           want{request: 512 * Mi, limit: 512 * Mi},
			err:            nil,
		},
		{
			name:           "failure from user setting exceeding the maximum memory",
			specMemory:     Memory{"onecontainer": int64Ptr(512 * Mi)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(128 * Mi),
			specMemMaximum: int64Ptr(128 * Mi),
			want:           want{},
			err:            ErrInvalidSetMemory,
		},
		{
			name:           "failure from Acornfile setting exceeding the maximum memory",
			specMemory:     Memory{},
			container:      Container{Memory: int64Ptr(512 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(128 * Mi),
			specMemMaximum: int64Ptr(128 * Mi),
			want:           want{},
			err:            ErrInvalidAcornMemory,
		},
		{
			name:           "failure from memory default exceeding the maximum memory",
			specMemory:     Memory{},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: int64Ptr(512 * Mi),
			specMemMaximum: int64Ptr(128 * Mi),
			want:           want{},
			err:            ErrInvalidDefaultMemory,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResources, err := MemoryToRequirements(tt.specMemory, tt.containerName, tt.container, tt.specMemDefault, tt.specMemMaximum)

			var wantedResources *corev1.ResourceRequirements
			if tt.want != (want{}) {
				wantedResources = &corev1.ResourceRequirements{
					Limits:   corev1.ResourceList{corev1.ResourceMemory: *resource.NewQuantity(tt.want.limit, resource.BinarySI)},
					Requests: corev1.ResourceList{corev1.ResourceMemory: *resource.NewQuantity(tt.want.request, resource.BinarySI)}}
			}

			assert.ErrorIs(t, err, tt.err)
			assert.EqualValues(t, wantedResources, actualResources)
		})
	}
}
