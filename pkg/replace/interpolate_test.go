package replace

import (
	"testing"
)

var testData = map[string]any{
	"a": map[string]any{
		"s": "string",
		"i": 42,
		"c": map[string]any{
			"x": 1,
		},
		"sl": []any{
			"x", 2,
		},
	},
}

func TestInterpolate(t *testing.T) {
	type args struct {
		data any
		s    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "String replace",
			args: args{
				data: testData,
				s:    "before @{a.s} after",
			},
			want: "before string after",
		},
		{
			name: "Number replace",
			args: args{
				data: testData,
				s:    "before @{a.i} after",
			},
			want: "before 42 after",
		},
		{
			name: "Map replace",
			args: args{
				data: testData,
				s:    "before @{a.c} after",
			},
			want: "before {\"x\":1} after",
		},
		{
			name: "Slice replace",
			args: args{
				data: testData,
				s:    "before @{a.sl[1]} after",
			},
			want: "before 2 after",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Interpolate(tt.args.data, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("Interpolate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Interpolate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
