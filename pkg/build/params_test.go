package build

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestParamsHelp(t *testing.T) {
	var (
		file = "testdata/params/herd.cue"
		cwd  = "testdata/params"
	)
	_, err := ParseParams(file, cwd, []string{
		"image-name",
		"--str=s",
		"--str-default=d",
		"-h",
		"--i=2",
		"--i-default=3",
		"--complex",
		"@testdata/params/test.cue",
	})
	assert.Equal(t, pflag.ErrHelp, err)
}

func TestParams(t *testing.T) {
	var (
		file = "testdata/params/herd.cue"
		cwd  = "testdata/params"
	)
	params, err := ParseParams(file, cwd, []string{
		"image-name",
		"--str=s",
		"--str-default=d",
		"--i=2",
		"--i-default=3",
		"--complex",
		"@testdata/params/test.cue",
	})
	if err != nil {
		t.Fatal(err)
	}

	def, err := ResolveAndParse(file, cwd)
	if err != nil {
		t.Fatal(err)
	}

	def, err = def.WithBuildParams(params)
	if err != nil {
		t.Fatal(err)
	}

	appSpec, err := def.AppSpec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "d3", appSpec.Containers["foo"].Image)
}
