package flagparams

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/rancher/wrangler/pkg/data/convert"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	params := v1.ParamSpec{
		Params: []v1.Param{
			{
				Name:        "intWithDefault",
				Description: "",
				Schema:      "*4 | int",
			},
			{
				Name:        "int",
				Description: "",
				Schema:      "int",
			},
			{
				Name:        "strWithDefault",
				Description: "",
				Schema:      "*s | string",
			},
			{
				Name:        "str",
				Description: "",
				Schema:      "string",
			},
			{
				Name:        "jsonFile",
				Description: "",
				Schema:      "string",
			},
			{
				Name:        "yamlFile",
				Description: "",
				Schema:      "string",
			},
			{
				Name:        "cueFile",
				Description: "",
				Schema:      "string",
			},
		},
	}

	flags := New("acorn.cue", &params)
	val, err := flags.Parse([]string{
		"--int", "1",
		"--int-with-default", "2",
		"--str", "a string",
		"--str-with-default", "b string",
		"--json-file", "@testdata/test.json",
		"--yaml-file", "@testdata/test.yaml",
		"--cue-file", "@testdata/test.cue",
	})
	if err != nil {
		t.Fatal(err)
	}

	normalizedVars := map[string]interface{}{}
	if err := convert.ToObj(val, &normalizedVars); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, map[string]interface{}{
		"int":            float64(1),
		"intWithDefault": float64(2),
		"str":            "a string",
		"strWithDefault": "b string",
		"jsonFile": map[string]interface{}{
			"value": "json",
		},
		"yamlFile": map[string]interface{}{
			"value": "yaml",
		},
		"cueFile": map[string]interface{}{
			"value": "cue",
		},
	}, normalizedVars)
}
