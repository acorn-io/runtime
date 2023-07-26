package events

import (
	"testing"
	"time"

	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/z"
	"github.com/stretchr/testify/assert"
)

func TestParseTimeBound(t *testing.T) {
	ts := internalv1.NowMicro()
	type args struct {
		raw string
		now internalv1.MicroTime
	}
	type want struct {
		parsed *internalv1.MicroTime
		err    assert.ErrorAssertionFunc
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "Go duration",
			args: args{
				raw: "1m",
				now: ts,
			},
			want: want{
				parsed: z.Pointer(internalv1.NewMicroTime(ts.Add(-1 * time.Minute))),
				err:    assert.NoError,
			},
		},
		{
			name: "Without Z",
			args: args{
				raw: "2023-01-09T12:32:00",
			},
			want: want{
				parsed: z.Pointer(internalv1.NewMicroTime(z.MustBe(time.Parse(time.RFC3339, "2023-01-09T12:32:00Z")))),
				err:    assert.NoError,
			},
		},
		{
			name: "With Z",
			args: args{
				raw: "2023-01-09T12:32:00Z",
			},
			want: want{
				parsed: z.Pointer(internalv1.NewMicroTime(z.MustBe(time.Parse(time.RFC3339, "2023-01-09T12:32:00Z")))),
				err:    assert.NoError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parseTimeBound(tt.args.raw, tt.args.now)
			if !tt.want.err(t, err) {
				return
			}
			assert.Equal(t, tt.want.parsed, parsed)
		})
	}
}
