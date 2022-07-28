package cue

import (
	"testing"

	"cuelang.org/go/cue"
	cue_mod "github.com/acorn-io/acorn/cue.mod"
	"github.com/acorn-io/acorn/schema"
	"github.com/stretchr/testify/assert"
)

var testAcornfile = []byte(`
import "github.com/acorn-io/acorn/schema/v1"

v1.#App & {
  containers: test: {
		image: "foo"
	}
}
`)

func newContext() *Context {
	return NewContext().
		WithNestedFS("schema", schema.Files).
		WithNestedFS("cue.mod", cue_mod.Files)
}

func TestDefaultContext(t *testing.T) {
	ctx := newContext()
	ctx = ctx.WithFile("test.cue", testAcornfile)
	v, err := ctx.Value()
	if err != nil {
		t.Fatal(err)
	}

	err = v.Validate(cue.Final())
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsConcrete())

	container := v.LookupPath(cue.ParsePath("containers.test"))
	if err != nil {
		t.Fatal(err)
	}
	defaultedContainer, _ := container.Default()
	image := defaultedContainer.LookupPath(cue.ParsePath("image"))

	s, err := image.String()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "foo", s)

	newV, _ := ctx.Value()
	assert.NotEqual(t, v, newV)
}
