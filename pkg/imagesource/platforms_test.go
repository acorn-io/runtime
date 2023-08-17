package imagesource

import (
	"context"
	"testing"

	"github.com/acorn-io/runtime/pkg/build"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestParamsHelp(t *testing.T) {
	var (
		file = "testdata/params/Acornfile"
		cwd  = "testdata/params"
	)
	_, _, err := NewImageSource("", file, []string{
		cwd,
		"image-name",
		"--str=s",
		"--str-default=d",
		"-h",
		"--i=2",
		"--i-default=3",
		"--complex",
		"@testdata/params/test.cue",
	}, nil, nil, false).GetAppDefinition(context.Background(), nil)
	assert.Equal(t, pflag.ErrHelp, err)
}

func TestParams(t *testing.T) {
	var (
		acornConfig string
		file        = "testdata/params/Acornfile"
		cwd         = "testdata/params"
	)
	_, params, err := NewImageSource(acornConfig, file, []string{
		cwd,
		"image-name",
		"--str=s",
		"--str-default=d",
		"--i=2",
		"--i-default=3",
		"--complex",
		"@testdata/params/test.cue",
	}, nil, nil, false).GetAppDefinition(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	def, err := build.ResolveAndParse(file)
	if err != nil {
		t.Fatal(err)
	}

	def, _, err = def.WithArgs(params, nil)
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "d3", appSpec.Containers["foo"].Image)
}
