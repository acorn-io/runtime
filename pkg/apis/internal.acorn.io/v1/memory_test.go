package v1

import (
	"testing"

	"github.com/acorn-io/z"
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
			specMemDefault: z.P(100 * Mi),
			specMemMaximum: new(int64),
			want:           z.P(100 * Mi),
			err:            nil,
		},
		{
			name:           "successful with setting from user",
			specMemory:     MemoryMap{"onecontainer": z.P(512 * Mi)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: new(int64),
			specMemMaximum: new(int64),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			name:           "successful with setting from Acornfile",
			specMemory:     MemoryMap{},
			container:      Container{Memory: z.P(512 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: new(int64),
			specMemMaximum: new(int64),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with user setting of 0 and max not being 0",
			specMemory:     MemoryMap{"onecontainer": new(int64)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: z.P(512 * Mi),
			specMemMaximum: z.P(512 * Mi),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with Acornfile setting of 0 and max not being 0",
			specMemory:     MemoryMap{},
			container:      Container{Memory: new(int64)},
			containerName:  "onecontainer",
			specMemDefault: z.P(512 * Mi),
			specMemMaximum: z.P(512 * Mi),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			// If memory is set to 0 but max is not 0, update setting to be maximum.
			name:           "successful with default setting of 0 and max not being 0",
			specMemory:     MemoryMap{},
			container:      Container{Memory: new(int64)},
			containerName:  "onecontainer",
			specMemDefault: new(int64),
			specMemMaximum: z.P(512 * Mi),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			name:           "successful overwrite of Acornfile with user setting",
			specMemory:     MemoryMap{"onecontainer": z.P(512 * Mi)},
			container:      Container{Memory: z.P(256 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: new(int64),
			specMemMaximum: new(int64),
			want:           z.P(512 * Mi),
			err:            nil,
		},
		{
			name:           "failure from user setting exceeding the maximum memory",
			specMemory:     MemoryMap{"onecontainer": z.P(512 * Mi)},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: z.P(128 * Mi),
			specMemMaximum: z.P(128 * Mi),
			want:           nil,
			err:            ErrInvalidSetMemory,
		},
		{
			name:           "failure from Acornfile setting exceeding the maximum memory",
			specMemory:     MemoryMap{},
			container:      Container{Memory: z.P(512 * Mi)},
			containerName:  "onecontainer",
			specMemDefault: z.P(128 * Mi),
			specMemMaximum: z.P(128 * Mi),
			want:           nil,
			err:            ErrInvalidAcornMemory,
		},
		{
			name:           "failure from memory default exceeding the maximum memory",
			specMemory:     MemoryMap{},
			container:      Container{},
			containerName:  "onecontainer",
			specMemDefault: z.P(512 * Mi),
			specMemMaximum: z.P(128 * Mi),
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
