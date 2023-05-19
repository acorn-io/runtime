package v1

import (
	"bytes"
	"encoding/json"

	"k8s.io/apimachinery/pkg/runtime"
)

type GenericMap map[string]interface{}

func (in GenericMap) MarshalJSON() ([]byte, error) {
	return json.Marshal((map[string]interface{})(in))
}

func translateObject(data interface{}) (ret interface{}, err error) {
	switch t := data.(type) {
	case map[string]interface{}:
		for k, v := range t {
			if t[k], err = translateObject(v); err != nil {
				return nil, err
			}
		}
	case []interface{}:
		for i, v := range t {
			if t[i], err = translateObject(v); err != nil {
				return nil, err
			}
		}
	case json.Number:
		i, err := t.Int64()
		if err == nil {
			return i, nil
		}
		return t.Float64()
	}
	return data, nil
}

func (in *GenericMap) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.UseNumber()
	if err := dec.Decode((*map[string]interface{})(in)); err != nil {
		return err
	}
	_, err := translateObject(*((*map[string]interface{})(in)))
	return err
}

func (in *GenericMap) DeepCopyInto(out *GenericMap) {
	*out = map[string]interface{}{}
	for k, v := range runtime.DeepCopyJSON(*in) {
		(*out)[k] = v
	}
}

func Mapify(obj any) (GenericMap, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	gm := make(GenericMap)
	return gm, gm.UnmarshalJSON(data)
}
