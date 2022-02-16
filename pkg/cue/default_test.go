package cue

import (
	"testing"

	"cuelang.org/go/cue"
	cue_mod "github.com/ibuildthecloud/herd/cue.mod"
	"github.com/ibuildthecloud/herd/schema"
	"github.com/stretchr/testify/assert"
)

var testHerdfile = []byte(`
import "github.com/ibuildthecloud/herd/schema/v1"

v1.#App & {
  containers: test: {
		image: "foo"
	}
}
`)

func TestTransform(t *testing.T) {
	ctx := newContext()
	ctx = ctx.WithFile("herd.cue", testHerdfile)
	v, err := ctx.Transform("github.com/ibuildthecloud/herd/schema/v1/transform/build")
	if err != nil {
		t.Fatal(err)
	}
	i := v.LookupPath(cue.ParsePath("containers.test.image"))
	if i.Err() != nil {
		t.Fatal(i.Err())
	}
	s, err := i.String()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", s)

	f := &struct {
		Containers map[string]struct {
			Image string `json:"image,omitempty"`
		} `json:"containers,omitempty"`
	}{}

	if err := v.Decode(f); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "foo", f.Containers["test"].Image)
}

func newContext() *Context {
	return NewContext().
		WithNestedFS("schema", schema.Files).
		WithNestedFS("cue.mod", cue_mod.Files)
}

func TestDefaultContext(t *testing.T) {
	ctx := newContext()
	ctx = ctx.WithFile("herd.cue", testHerdfile)
	v, err := ctx.Value()
	if err != nil {
		t.Fatal(err)
	}

	err = v.Validate(cue.Final())
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, v.IsConcrete())

	image := v.LookupPath(cue.ParsePath("containers.test.image"))
	if err != nil {
		t.Fatal(err)
	}

	s, err := image.String()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "foo", s)

	newV, _ := ctx.Value()
	assert.NotEqual(t, v, newV)
}
