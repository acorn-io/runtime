package secrets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInflateRanges(t *testing.T) {
	for _, tt := range []struct {
		name string
		want string
	}{
		{
			name: "",
			want: "",
		},
		{
			name: "-",
			want: "-",
		},
		{
			name: "0-a",
			want: "-0a",
		},
		{
			name: "A-",
			want: "-A",
		},
		{
			name: "A-A",
			want: "A",
		},
		{
			name: "A-Z",
			want: "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name: "Z-A",
			want: "-AZ",
		},
		{
			name: "A-z",
			want: "-Az",
		},
		{
			name: "z-A",
			want: "-Az",
		},
		{
			name: "a-Z",
			want: "-Za",
		},
		{
			name: "Z-a",
			want: "-Za",
		},
		{
			name: "z-a",
			want: "-az",
		},
		{
			name: "a-z",
			want: "abcdefghijklmnopqrstuvwxyz",
		},
		{
			name: "0-9",
			want: "0123456789",
		},
		{
			name: "9-0",
			want: "-09",
		},
		{
			name: "0-9A-Z",
			want: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name: "A-Z0-9",
			want: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ",
		},
		{
			name: "A-Za-z0-9",
			want: "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
		},
		{
			name: "!#$%&*+-0123456789=ABCDEFGHIJKLMNOPQRSTUVWXYZ^_abcdefghijklmnopqrstuvwxyz",
			want: "!#$%&*+-0123456789=ABCDEFGHIJKLMNOPQRSTUVWXYZ^_abcdefghijklmnopqrstuvwxyz",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, inflateRanges(tt.name))
		})
	}
}
