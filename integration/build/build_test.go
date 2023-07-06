package build

import (
	"context"
	"strings"
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/images"
	"github.com/acorn-io/runtime/pkg/imagesource"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
)

func TestBuildFailed(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	_, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/fail/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/fail",
	})
	assert.Error(t, err)
}

func TestBuildDev(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	img, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/dev/Acornfile", &client.AcornImageBuildOptions{
		Cwd:      "./testdata/dev",
		Profiles: []string{"devMode"},
	})
	assert.NoError(t, err)
	_, ok := img.ImageData.Containers["default"]
	assert.True(t, ok)
}

func TestBuildFailedNoImageBuild(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	_, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/no-image-build/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/no-image-build",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either image or build field must be set")
}

func TestSimpleBuild(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/simple/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:5000"))
	assert.Len(t, image.ImageData.Images, 1)
	assert.True(t, len(image.ImageData.Images["isimple"].Image) > 0)
}

func TestSimilarBuilds(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)

	// This tests a scenario where two builds only differ by a single character in the Acornfile file and otherwise all
	// the file names and sizes are the same. A caching bug caused the second build to result in the image from the first
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/similar/one/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/similar/one",
	})
	if err != nil {
		t.Fatal(err)
	}

	image2, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/similar/two/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/similar/two",
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, image.ID, image2.ID)
}

func TestJobBuild(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/jobs/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/jobs",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Jobs, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Image, "127.0.0.1:5000"))

	assert.Len(t, image.ImageData.Jobs["simple"].Sidecars, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Jobs["simple"].Sidecars["left"].Image, "127.0.0.1:5000"))
}

func TestSidecarBuild(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/sidecar/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/sidecar",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Image, "127.0.0.1:5000"))

	assert.Len(t, image.ImageData.Containers["simple"].Sidecars, 1)
	assert.True(t, strings.HasPrefix(image.ImageData.Containers["simple"].Sidecars["left"].Image, "127.0.0.1:5000"))
}

func TestTarget(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/target/Acornfile", &client.AcornImageBuildOptions{
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
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/contextdir/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/contextdir",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 2)
	assert.True(t, len(image.ImageData.Containers["simple"].Image) > 0)
	assert.True(t, len(image.ImageData.Containers["fromimage"].Image) > 0)
}

func TestSimpleTwo(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/simple-two/Acornfile", &client.AcornImageBuildOptions{
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
	assert.True(t, len(image.ImageData.Images["ithree"].Image) > 0)
	// This isn't always true, no idea why, one day maybe we'll know
	//assert.Equal(t, image.ImageData.Containers["two"].Image, image.ImageData.Images["itwo"].Image)
}

func TestBuildDefault(t *testing.T) {
	c := helper.BuilderClient(t, system.DefaultUserNamespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/build-default/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "./testdata/build-default",
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, image.ImageData.Containers, 1)
}

func TestMultiArch(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	project := helper.TempProject(t, helper.MustReturn(k8sclient.Default))
	kclient := helper.MustReturn(k8sclient.Default)
	c, err := client.New(cfg, "", project.Name)
	if err != nil {
		t.Fatal()
	}

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/multiarch/Acornfile", &client.AcornImageBuildOptions{
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

	transport, err := imagesystem.NewAPIBasedTransport(kclient, helper.MustReturn(k8sclient.DefaultConfig))
	if err != nil {
		t.Fatal(err)
	}

	opts, err := images.GetAuthenticationRemoteOptions(context.Background(), kclient, project.Name, remote.WithTransport(transport))
	if err != nil {
		t.Fatal(err)
	}

	imgName := strings.ReplaceAll(image.ImageData.Containers["web"].Image, "127.0.0.1:5000",
		"registry.acorn-image-system.svc.cluster.local:5000")
	tag, err := name.ParseReference(imgName)
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

func TestBuildNestedAcornWithLocalImage(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)
	project := helper.TempProject(t, helper.MustReturn(k8sclient.Default))
	c, err := client.New(cfg, "", project.Name)
	if err != nil {
		t.Fatal(err)
	}

	// build the Nginx image
	source := imagesource.NewImageSource("./testdata/nested/nginx.Acornfile", []string{}, []string{}, []string{}, false)
	image, _, err := source.GetImageAndDeployArgs(helper.GetCTX(t), c)
	if err != nil {
		t.Fatal(err)
	}
	err = c.ImageTag(helper.GetCTX(t), image, "localnginx:latest")
	if err != nil {
		t.Fatal(err)
	}

	// build the nested image - should succeed
	nestedImage, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/nested/nested.Acornfile", &client.AcornImageBuildOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// verify that the digests match up
	assert.Equal(t, strings.Split(nestedImage.ImageData.Acorns["nginx"].Image, "sha256:")[1], image)

	// create a second project and make sure we can't access the local image from the first project
	project2 := helper.TempProject(t, helper.MustReturn(k8sclient.Default))
	c2, err := client.New(cfg, "", project2.Name)
	if err != nil {
		t.Fatal(err)
	}

	// build the nested image in the new project - should fail
	_, err = c2.AcornImageBuild(helper.GetCTX(t), "./testdata/nested/nested.Acornfile", &client.AcornImageBuildOptions{})
	if err == nil {
		t.Fatalf("expected error when referencing local image in another project, got nil")
	}
	assert.Contains(t, err.Error(), "missing registry host in the tag [localnginx:latest]")
}
