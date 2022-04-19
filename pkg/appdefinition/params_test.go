package appdefinition

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParamSpec(t *testing.T) {
	herdCue := `
params: build: {
  // Description of a string param
  foo: string

  // Two line Description of an int
  // Description of an int with default
//
  bar: int | *4
// This is dropped

// Complex  value 
  complex: {
    foo: string
  }
}
`
	def, err := NewAppDefinition([]byte(herdCue))
	if err != nil {
		t.Fatal(err)
	}

	spec, err := def.BuildParams()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", spec.Params[0].Name)
	assert.Equal(t, "string", spec.Params[0].Schema)
	assert.Equal(t, "Description of a string param", spec.Params[0].Description)

	assert.Equal(t, "bar", spec.Params[1].Name)
	assert.Equal(t, "*4 | int", spec.Params[1].Schema)
	assert.Equal(t, "Two line Description of an int\nDescription of an int with default", spec.Params[1].Description)

	assert.Equal(t, "complex", spec.Params[2].Name)
	assert.Equal(t, "{\n\tfoo: string\n}", spec.Params[2].Schema)
	assert.Equal(t, "Complex  value", spec.Params[2].Description)
}
