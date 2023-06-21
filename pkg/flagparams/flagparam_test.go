package flagparams

import (
	"runtime"
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
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
				Name:        "intWithDefaultAllowZero",
				Description: "",
				Schema:      "*4 | int",
			},
			{
				Name:        "intShouldNotBeInParamMap",
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
				Name:        "strShouldNotBeInParamMap",
				Description: "",
				Schema:      "*\"\" | string",
			},
			{
				Name:        "strWithEmptyValue",
				Description: "",
				Schema:      "*s | string",
			},
			{
				Name:        "jsonFile",
				Description: "",
				Schema:      "complex",
			},
			{
				Name:        "yamlFile",
				Description: "",
				Schema:      "complex",
			},
			{
				Name:        "cueFile",
				Description: "",
				Schema:      "complex",
			},
			{
				Name:        "anEmptyComplex",
				Description: "",
				Schema:      "complex",
			},
			{
				Name:        "cueString",
				Description: "",
				Schema:      "string",
			},
			{
				Name:        "abool",
				Description: "",
				Schema:      "bool",
			},
			{
				Name:        "aDefaultBool",
				Description: "",
				Schema:      "*false | bool",
			},
			{
				Name:        "passAFalseBool",
				Description: "",
				Schema:      "*true| bool",
			},
		},
	}

	flags := New("Acornfile", &params)
	val, err := flags.Parse([]string{
		"--int", "1",
		"--int-with-default", "2",
		"--int-with-default-allow-zero", "0",
		"--str", "a string",
		"--str-with-default", "b string",
		"--str-with-empty-value", "",
		"--json-file", "@testdata/test.json",
		"--yaml-file", "@testdata/test.yaml",
		"--cue-file", "@testdata/test.cue",
		"--cue-string", "@testdata/test.cue",
		"--an-empty-complex", "",
		"--abool",
		"--a-default-bool",
		"--pass-a-false-bool=false",
	})
	if err != nil {
		t.Fatal(err)
	}

	normalizedVars := map[string]any{}
	if err := convert.ToObj(val, &normalizedVars); err != nil {
		t.Fatal(err)
	}

	cuestring := "{\n\tvalue: \"cue\"\n}\n"
	if runtime.GOOS == "windows" {
		cuestring = "{\r\n\tvalue: \"cue\"\r\n}\r\n"
	}

	assert.Equal(t, map[string]any{
		"int":                     float64(1),
		"intWithDefault":          float64(2),
		"intWithDefaultAllowZero": float64(0),
		"str":                     "a string",
		"strWithDefault":          "b string",
		"strWithEmptyValue":       "",
		"jsonFile": map[string]any{
			"value": "json",
		},
		"yamlFile": map[string]any{
			"value": "yaml",
		},
		"cueFile": map[string]any{
			"value": "cue",
		},
		"cueString":      cuestring,
		"anEmptyComplex": "",
		"abool":          true,
		"aDefaultBool":   true,
		"passAFalseBool": false,
	}, normalizedVars)
}
