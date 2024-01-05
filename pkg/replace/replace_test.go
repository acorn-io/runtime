package replace

import (
	"fmt"
	"testing"
)

func countReplacer() func(string) (string, bool, error) {
	i := 0
	return func(s string) (string, bool, error) {
		i++
		return fmt.Sprintf("%s:%d", s, i), true, nil
	}
}

func TestReplace(t *testing.T) {
	type args struct {
		s          string
		startToken string
		endToken   string
		replace    func(string) (string, bool, error)
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "skip @@",
			args: args{
				s:          "@@{@{first}",
				startToken: "@{",
				endToken:   "}",
				replace:    countReplacer(),
			},
			want: "@@{first:1",
		},
		{
			name: "one replace",
			args: args{
				s:          "@{first}",
				startToken: "@{",
				endToken:   "}",
				replace:    countReplacer(),
			},
			want: "first:1",
		},
		{
			name: "two replace",
			args: args{
				s:          "@{first}@{second}",
				startToken: "@{",
				endToken:   "}",
				replace:    countReplacer(),
			},
			want: "first:1second:2",
		},
		{
			name: "two replace with content around",
			args: args{
				s:          "start @{first} middle @{second} end",
				startToken: "@{",
				endToken:   "}",
				replace:    countReplacer(),
			},
			want: "start first:1 middle second:2 end",
		},
		{
			name: "empty var",
			args: args{
				s:          "start@{}end",
				startToken: "@{",
				endToken:   "}",
				replace:    countReplacer(),
			},
			want: "start:1end",
		},
		{
			name: "no replace var",
			args: args{
				s:          "start@{inner}end",
				startToken: "@{",
				endToken:   "}",
				replace: func(s string) (string, bool, error) {
					return "", false, nil
				},
			},
			want: "start@{inner}end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Replace(tt.args.s, tt.args.startToken, tt.args.endToken, tt.args.replace)
			if (err != nil) != tt.wantErr {
				t.Errorf("Replace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Replace() got = %v, want %v", got, tt.want)
			}
		})
	}
}
