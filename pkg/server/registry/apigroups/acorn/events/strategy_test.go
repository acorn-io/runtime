package events

import (
	"testing"
	"time"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
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

func TestQueryFilter(t *testing.T) {
	ts := internalv1.NowMicro()
	tests := []struct {
		name  string
		query query
		args  []apiv1.Event
		want  []apiv1.Event
	}{
		{
			name: "Tail less than length",
			query: query{
				tail: 1,
			},
			args: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
			want: []apiv1.Event{
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
		},
		{
			name: "Tail more than length",
			query: query{
				tail: 3,
			},
			args: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
			want: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
		},
		{
			name: "Tail since",
			query: query{
				tail:  2,
				since: z.Pointer(internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))),
			},
			args: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
			want: []apiv1.Event{
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
		},
		{
			name: "Tail until",
			query: query{
				tail:  2,
				until: z.Pointer(internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))),
			},
			args: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
				{Observed: internalv1.NewMicroTime(ts.Add(2 * time.Microsecond))},
			},
			want: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
			},
		},
		{
			name: "Tail window",
			query: query{
				tail:  2,
				since: z.Pointer(internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))),
				until: z.Pointer(internalv1.NewMicroTime(ts.Add(3 * time.Microsecond))),
			},
			args: []apiv1.Event{
				{Observed: ts},
				{Observed: internalv1.NewMicroTime(ts.Add(1 * time.Microsecond))},
				{Observed: internalv1.NewMicroTime(ts.Add(2 * time.Microsecond))},
				{Observed: internalv1.NewMicroTime(ts.Add(3 * time.Microsecond))},
				{Observed: internalv1.NewMicroTime(ts.Add(4 * time.Microsecond))},
			},
			want: []apiv1.Event{
				{Observed: internalv1.NewMicroTime(ts.Add(2 * time.Microsecond))},
				{Observed: internalv1.NewMicroTime(ts.Add(3 * time.Microsecond))},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.ElementsMatch(t, tt.want, tt.query.filter(tt.args...))
		})
	}
}
