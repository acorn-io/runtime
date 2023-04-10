package publicname

import "testing"

func TestSplit(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		prefix string
		suffix string
	}{
		{
			name:   "no child",
			input:  "foo",
			prefix: "",
			suffix: "foo",
		},
		{
			name:   "one child",
			input:  "foo.bar",
			prefix: "foo",
			suffix: "bar",
		},
		{
			name:   "two child",
			input:  "foo.bar.baz",
			prefix: "foo.bar",
			suffix: "baz",
		},
		{
			name:   "start with .",
			input:  ".bar.baz",
			prefix: "",
			suffix: ".bar.baz",
		},
		{
			name:   "start end with .",
			input:  "bar.baz.",
			prefix: "",
			suffix: "bar.baz.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := Split(tt.input)
			if got != tt.prefix {
				t.Errorf("Split() got = %v, want %v", got, tt.prefix)
			}
			if got1 != tt.suffix {
				t.Errorf("Split() got1 = %v, want %v", got1, tt.suffix)
			}
		})
	}
}
