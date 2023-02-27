package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateMemory(t *testing.T) {
	const Mi = int64(1048576)
	tests := []struct {
		name           string
		specMemory     MemoryMap
		container      Container
		containerName  string
		specMemDefault *int64
		specMemMaximum *int64
		want           *int64
		err            error
	}{
		{
			name:           "successful with default",
			specMemory:     MemoryMap{},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{100 * Mi}[0],
			specMemMaximum: &[]int64{0}[0],
			want:           &[]int64{100 * Mi}[0],
			err:            nil,
		},
		{
			name:           "successful with setting from user",
			specMemory:     MemoryMap{"onecontainer": &[]int64{512 * Mi}[0]},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{0}[0],
			specMemMaximum: &[]int64{0}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			name:           "successful with setting from Acornfile",
			specMemory:     MemoryMap{},
			container:      Container{Memory: &[]int64{512 * Mi}[0]},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{0}[0],
			specMemMaximum: &[]int64{0}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with user setting of 0 and max not being 0",
			specMemory:     MemoryMap{"onecontainer": &[]int64{0}[0]},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{512 * Mi}[0],
			specMemMaximum: &[]int64{512 * Mi}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with Acornfile setting of 0 and max not being 0",
			specMemory:     MemoryMap{},
			container:      Container{Memory: &[]int64{0}[0]},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{512 * Mi}[0],
			specMemMaximum: &[]int64{512 * Mi}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with default setting of 0 and max not being 0",
			specMemory:     MemoryMap{},
			container:      Container{Memory: &[]int64{0}[0]},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{0}[0],
			specMemMaximum: &[]int64{512 * Mi}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			name:           "successful overwrite of Acornfile with user setting",
			specMemory:     MemoryMap{"onecontainer": &[]int64{512 * Mi}[0]},
			container:      Container{Memory: &[]int64{256 * Mi}[0]},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{0}[0],
			specMemMaximum: &[]int64{0}[0],
			want:           &[]int64{512 * Mi}[0],
			err:            nil,
		},
		{
			name:           "failure from user setting exceeding the maximum memory",
			specMemory:     MemoryMap{"onecontainer": &[]int64{512 * Mi}[0]},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{128 * Mi}[0],
			specMemMaximum: &[]int64{128 * Mi}[0],
			want:           nil,
			err:            ErrInvalidSetMemory,
		},
		{
			name:           "failure from Acornfile setting exceeding the maximum memory",
			specMemory:     MemoryMap{},
			container:      Container{Memory: &[]int64{512 * Mi}[0]},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{128 * Mi}[0],
			specMemMaximum: &[]int64{128 * Mi}[0],
			want:           nil,
			err:            ErrInvalidAcornMemory,
		},
		{
			name:           "failure from memory default exceeding the maximum memory",
			specMemory:     MemoryMap{},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: &[]int64{512 * Mi}[0],
			specMemMaximum: &[]int64{128 * Mi}[0],
			want:           nil,
			err:            ErrInvalidDefaultMemory,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ValidateMemory(tt.specMemory, tt.containerName, tt.container, tt.specMemDefault, tt.specMemMaximum)

			if tt.err != nil {
				assert.ErrorIs(t, err, tt.err)
				return
			}

			if tt.want != nil {
				assert.EqualValues(t, *tt.want, actual.Value())
			}
		})
	}
}
