package replace

import (
	"context"
	"encoding/json"

	"github.com/acorn-io/aml"
	"github.com/acorn-io/aml/pkg/eval"
	"github.com/acorn-io/aml/pkg/value"
)

func Interpolate(data any, s string) (string, error) {
	v, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var val value.Value
	if err := aml.Unmarshal(v, &val); err != nil {
		return "", err
	}
	return Replace(s, "@{", "}", func(s string) (string, bool, error) {
		out := json.RawMessage{}
		err = aml.Unmarshal([]byte(s), &out, aml.DecoderOption{
			SourceName: "inline",
			GlobalsLookup: func(_ context.Context, key string, _ eval.Scope) (value.Value, bool, error) {
				return value.Lookup(val, value.NewValue(key))
			},
		})
		if len(out) > 0 && out[0] == '"' {
			var s string
			err := json.Unmarshal(out, &s)
			if err != nil {
				return "", false, err
			}
			return s, true, nil
		}
		return string(out), true, nil
	})
}
