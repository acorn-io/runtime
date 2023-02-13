package data

import (
	"bytes"
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func ToMapInterface(obj interface{}) (map[string]interface{}, error) {
	if m, ok := obj.(map[string]interface{}); ok {
		return m, nil
	}

	if unstr, ok := obj.(*unstructured.Unstructured); ok {
		return unstr.Object, nil
	}

	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	dec := json.NewDecoder(bytes.NewBuffer(b))
	dec.UseNumber()
	return result, dec.Decode(&result)
}
