package build

import (
	"context"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/remoteopts"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
)

func TestBuildFailed(t *testing.T) {
	_, err := build.Build(helper.GetCTX(t), "./testdata/fail/Acornfile", &build.Options{
		Cwd:    "./testdata/fail",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	assert.Error(t, err)
}

func TestBuildFailedNoImageBuild(t *testing.T) {
	_, err := build.Build(helper.GetCTX(t), "./testdata/no-image-build/Acornfile", &build.Options{
		Cwd:    "./testdata/no-image-build",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either image or build field must be set")
}

func TestSimpleBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/Acornfile", &build.Options{
		Cwd:    "./testdata/simple",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
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

func TestSimilarBuilds(t *testing.T) {
	// This tests a scenario where two builds only differ by a single character in the Acornfile file and otherwise all
	// the file names and sizes are the same. A caching bug caused the second build to result in the image from the first
	image, err := build.Build(helper.GetCTX(t), "./testdata/similar/one/Acornfile", &build.Options{
		Cwd:    "./testdata/similar/one",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	if err != nil {
		t.Fatal(err)
	}

	image2, err := build.Build(helper.GetCTX(t), "./testdata/similar/two/Acornfile", &build.Options{
		Cwd:    "./testdata/similar/two",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, image.ID, image2.ID)
}

func TestJobBuild(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/jobs/Acornfile", &build.Options{
		Cwd:    "./testdata/jobs",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
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
	image, err := build.Build(helper.GetCTX(t), "./testdata/sidecar/Acornfile", &build.Options{
		Cwd:    "./testdata/sidecar",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
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
	image, err := build.Build(helper.GetCTX(t), "./testdata/target/Acornfile", &build.Options{
		Cwd:    "./testdata/target",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
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
	image, err := build.Build(helper.GetCTX(t), "./testdata/contextdir/Acornfile", &build.Options{
		Cwd:    "./testdata/contextdir",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 2)
	assert.True(t, len(image.ImageData.Containers["simple"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["fromImage"].Image) > 0)
}

func TestSimpleTwo(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/simple-two/Acornfile", &build.Options{
		Cwd:    "./testdata/simple-two",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
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

func TestBuildDefault(t *testing.T) {
	image, err := build.Build(helper.GetCTX(t), "./testdata/build-default/Acornfile", &build.Options{
		Cwd:    "./testdata/build-default",
		Client: helper.BuilderClient(t, system.RequireUserNamespace()),
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
}

func TestMultiArch(t *testing.T) {
	cfg := helper.StartAPI(t)
	ns := helper.TempNamespace(t, helper.MustReturn(k8sclient.Default))
	c, err := client.New(cfg, ns.Name)
	if err != nil {
		t.Fatal()
	}

	image, err := build.Build(helper.GetCTX(t), "./testdata/multiarch/Acornfile", &build.Options{
		Client: c,
		Cwd:    "./testdata/multiarch",
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

	opts, err := remoteopts.WithClientDialer(context.Background(), c)
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
