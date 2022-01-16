package build

import (
	"context"
	"testing"

	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/stretchr/testify/assert"
)

func getCtx(t *testing.T) context.Context {
	deadline, ok := t.Deadline()
	if !ok {
		return context.Background()
	}
	ctx, _ := context.WithDeadline(context.Background(), deadline)
	return ctx
}

func TestSimpleBuild(t *testing.T) {
	image, err := build.Build(getCtx(t), "./testdata/simple/herd.cue", &build.Opts{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, len(image.ImageData.Containers["simple"].Image) > 0)
}

func TestSimpleTwo(t *testing.T) {
	image, err := build.Build(getCtx(t), "./testdata/simple-two/herd.cue", &build.Opts{
		Cwd: "./testdata/simple-two",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 2)
	assert.True(t, len(image.ImageData.Containers["one"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["two"].Image) > 0)
}
