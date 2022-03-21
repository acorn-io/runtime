package v1

import (
	"encoding/json"

	"github.com/rancher/wrangler/pkg/data/convert"
)

type GenericMap map[string]interface{}

func (in GenericMap) MarshalJSON() ([]byte, error) {
	return json.Marshal((map[string]interface{})(in))
}

func (in *GenericMap) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[string]interface{})(in))
}

func (in *GenericMap) DeepCopyInto(out *GenericMap) {
	if err := convert.ToObj(in, (*map[string]interface{})(out)); err != nil {
		panic(err)
	}
}
