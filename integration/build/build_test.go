package build

import (
	"context"
	"strings"
	"testing"

	"github.com/ibuildthecloud/herd/integration/helper"
	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/ibuildthecloud/herd/pkg/build/buildkit"
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/stretchr/testify/assert"
)

func TestBuildFailed(t *testing.T) {
	_, err := build.Build(helper.GetCTX(t), "./testdata/fail/herd.cue", &build.Options{
		Cwd: "./testdata/fail",
	})
	assert.Error(t, err)
}

func TestSimpleBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/herd.cue", &build.Options{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:"))
	assert.False(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:5000"))
	assert.Len(t, image.ImageData.Images, 1)
	assert.True(t, len(image.ImageData.Images["isimple"].Image) > 0)
}

func TestTarget(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/target/herd.cue", &build.Options{
		Cwd: "./testdata/target",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, len(image.ImageData.Containers["simple"].Image) > 0)
	assert.Len(t, image.ImageData.Images, 1)
	assert.True(t, len(image.ImageData.Images["isimple"].Image) > 0)
}

func TestSimpleTwo(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple-two/herd.cue", &build.Options{
		Cwd: "./testdata/simple-two",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 2)
	assert.True(t, len(image.ImageData.Containers["one"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["two"].Image) > 0)
	assert.Len(t, image.ImageData.Images, 2)
	assert.True(t, len(image.ImageData.Images["ione"].Image) > 0)
	assert.True(t, len(image.ImageData.Images["itwo"].Image) > 0)
	assert.Equal(t, image.ImageData.Containers["two"].Image, image.ImageData.Images["itwo"].Image)
}

func Test_GetBuildkitDialer(t *testing.T) {
	c, err := client.Default()
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(helper.GetCTX(t))
	defer cancel()
	port, _, err := buildkit.GetBuildkitDialer(ctx, c)
	assert.Nil(t, err)
	assert.True(t, port > 30000)
}
