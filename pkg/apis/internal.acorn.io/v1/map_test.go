package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericMapRoundtrip(t *testing.T) {
	type object struct {
		Name    string      `json:"name"`
		Details *GenericMap `json:"gm,omitempty"`
	}
	in := object{
		Name: "my-object",
		Details: NewGenericMap(map[string]any{
			"fruit": "apple",
			"count": int64(1),
		}),
	}

	marshaled, err := json.Marshal(in)
	require.NoError(t, err)

	var out object
	require.NoError(t, json.Unmarshal(marshaled, &out))
	require.Equal(t, in, out)
}

func TestMapify(t *testing.T) {
	type nested struct {
		Fruit string `json:"fruit"`
		Count int    `json:"count"`
	}

	gm, err := Mapify(nested{
		Fruit: "apple",
		Count: 1,
	})
	require.NoError(t, err)
	require.Equal(t, GenericMap{
		Data: map[string]any{
			"fruit": "apple",
			"count": int64(1),
		},
	}, gm)
}
