package cue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue/cuecontext"
	"sigs.k8s.io/yaml"
)

func ReadCUE(file string) ([]byte, error) {
	fileData, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(file)
	if ext == ".yaml" || ext == ".json" {
		data := map[string]any{}
		err := yaml.Unmarshal(fileData, &data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", file, err)
		}
		fileData, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("converting %s: %w", file, err)
		}
	}

	return fileData, nil
}

func UnmarshalFile(file string, obj any) error {
	fileData, err := ReadCUE(file)
	if err != nil {
		return err
	}
	ctx := cuecontext.New()
	jsonBytes, err := ctx.CompileString(string(fileData)).MarshalJSON()
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, obj)
}
