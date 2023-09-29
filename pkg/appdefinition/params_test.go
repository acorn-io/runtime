package appdefinition

import (
	"encoding/json"
	"testing"

	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestParamTypes(t *testing.T) {
	acornCue := `
args: {
	s: "string"
	b: true
	i: 4
	f: 5.0
	e: enum("hi", "bye") || default "hi"
	a: [string] || default ["hi"]
	o: object || default {}
}
`
	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := def.ToParamSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "string", string(spec.Args[0].Type.Kind))
	assert.Equal(t, "bool", string(spec.Args[1].Type.Kind))
	assert.Equal(t, "number", string(spec.Args[2].Type.Kind))
	assert.Equal(t, "number", string(spec.Args[3].Type.Kind))
	assert.Equal(t, "string", string(spec.Args[4].Type.Kind))
	assert.Equal(t, "array", string(spec.Args[5].Type.Kind))
	assert.Equal(t, "object", string(spec.Args[6].Type.Kind))
}

func TestParamSpec(t *testing.T) {
	acornCue := `
args: {
  // Description of a string param
  foo: "x"

  // Two line Description of an int
  // Description of an int with default
//
  bar: int || default 4
// This is dropped

// Complex  value 
  complex?: {
    foo: "hi"
  }
}
`
	def, err := NewAppDefinition([]byte(acornCue))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := def.ToParamSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", spec.Args[0].Name)
	assert.Equal(t, "Description of a string param", spec.Args[0].Description)

	assert.Equal(t, "bar", spec.Args[1].Name)
	assert.Equal(t, "Two line Description of an int\nDescription of an int with default", spec.Args[1].Description)

	assert.Equal(t, "complex", spec.Args[2].Name)
	assert.Equal(t, "Complex  value", spec.Args[2].Description)
}

func TestJSONFloatParsing(t *testing.T) {
	data := []byte(`
args: {
	replicas: 1
}

profiles: {
	prod: {
		replicas: 2
	}
}

containers: {
	web: {
		image: "ghcr.io/acorn-io/images-mirror/nginx:latest"
		scale: args.replicas
	}
}`)

	appDef, err := NewAppDefinition(data)
	if err != nil {
		t.Fatal(err)
	}

	params := v1.GenericMap{}
	err = json.Unmarshal([]byte(`{"replicas": 3}`), &params)
	if err != nil {
		t.Fatal(err)
	}

	appDef = appDef.WithArgs(params.GetData(), []string{"prod"})

	appSpec, err := appDef.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int32(3), *appSpec.Containers["web"].Scale)
}
