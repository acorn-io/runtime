package imagesource

import (
	"context"
	"testing"

	"github.com/acorn-io/runtime/pkg/build"
	"github.com/stretchr/testify/assert"
)

func TestParams(t *testing.T) {
	var (
		acornConfig string
		file        = "testdata/params/Acornfile"
		cwd         = "testdata/params"
	)
	_, params, _, err := NewImageSource(acornConfig, file, "", []string{
		cwd,
		"image-name",
		"--str=s",
		"--strDefault=d",
		"--i=2",
		"--iDefault=3",
		"--complex",
		"@testdata/params/test.cue",
	}, nil, false).GetAppDefinition(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	def, err := build.ResolveAndParse(file)
	if err != nil {
		t.Fatal(err)
	}

	def = def.WithArgs(params, nil)

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "d3", appSpec.Containers["foo"].Image)
}
