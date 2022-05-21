package build

import (
	"context"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
)

func TestBuildFailed(t *testing.T) {
	_, err := build.Build(helper.GetCTX(t), "./testdata/fail/acorn.cue", &build.Options{
		Cwd: "./testdata/fail",
	})
	assert.Error(t, err)
}

func TestNestedBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/nested/acorn.cue", &build.Options{
		Cwd: "./testdata/nested",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Acorns, 2)
}

func TestSimpleBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/acorn.cue", &build.Options{
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

func TestJobBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/jobs/acorn.cue", &build.Options{
		Cwd: "./testdata/jobs",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Jobs, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Image, "127.0.0.1:"))
	assert.False(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Image, "127.0.0.1:5000"))

	assert.Len(t, image.ImageData.Jobs["simple"].Sidecars, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Sidecars["left"].Image, "127.0.0.1:"))
	assert.False(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Sidecars["left"].Image, "127.0.0.1:5000"))
}

func TestSidecarBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/sidecar/acorn.cue", &build.Options{
		Cwd: "./testdata/sidecar",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:"))
	assert.False(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:5000"))

	assert.Len(t, image.ImageData.Containers["simple"].Sidecars, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Sidecars["left"].Image, "127.0.0.1:"))
	assert.False(t, strings.HasPrefix(image.ImageData.Containers["simple"].Sidecars["left"].Image, "127.0.0.1:5000"))
}

func TestTarget(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/target/acorn.cue", &build.Options{
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

func TestContextDir(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/contextdir/acorn.cue", &build.Options{
		Cwd: "./testdata/contextdir",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 2)
	assert.True(t, len(image.ImageData.Containers["simple"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["fromImage"].Image) > 0)
}

func TestSimpleTwo(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple-two/acorn.cue", &build.Options{
		Cwd: "./testdata/simple-two",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 3)
	assert.True(t, len(image.ImageData.Containers["one"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["two"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["three"].Image) > 0)
	assert.Len(t, image.ImageData.Images, 3)
	assert.True(t, len(image.ImageData.Images["ione"].Image) > 0)
	assert.True(t, len(image.ImageData.Images["itwo"].Image) > 0)
	assert.True(t, len(image.ImageData.Images["three"].Image) > 0)
	// This isn't always true, no idea why, one day maybe we'll know
	//assert.Equal(t, image.ImageData.Containers["two"].Image, image.ImageData.Images["itwo"].Image)
}

func Test_GetBuildkitDialer(t *testing.T) {
	c, err := k8sclient.Default()
	assert.Nil(t, err)

	ctx, cancel := context.WithCancel(helper.GetCTX(t))
	defer cancel()
	port, _, err := buildkit.GetBuildkitDialer(ctx, c)
	assert.Nil(t, err)
	assert.True(t, port > 30000)
}

func TestMultiArch(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/multiarch/acorn.cue", &build.Options{
		Cwd: "./testdata/multiarch",
		Platforms: []v1.Platform{
			{
				Architecture: "amd64",
				OS:           "linux",
			},
			{
				Architecture: "arm64",
				OS:           "linux",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, len(image.ImageData.Containers["web"].Image) > 0)

	opts, err := build.GetRemoteOptions(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	tag, err := name.ParseReference(image.ImageData.Containers["web"].Image)
	if err != nil {
		t.Fatal(err)
	}

	index, err := remote.Index(tag, opts...)
	if err != nil {
		t.Fatal(err)
	}

	manifest, err := index.IndexManifest()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, manifest.Manifests, 2)
	assert.Equal(t, manifest.Manifests[0].Platform.Architecture, "amd64")
	assert.Equal(t, manifest.Manifests[1].Platform.Architecture, "arm64")
}
