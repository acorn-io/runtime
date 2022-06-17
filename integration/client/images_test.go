package client

import (
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	"github.com/acorn-io/acorn/pkg/build"
	"github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/stretchr/testify/assert"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestImageListGetDelete(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := newImage(t, ns.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	newImage, err := c.ImageGet(ctx, image.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, image.Name, newImage.Name)

	delImage, err := c.ImageDelete(ctx, image.Name)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, newImage.Name, delImage.Name)

	_, err = c.ImageGet(ctx, image.Name)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestImageTagMove(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := newImage(t, ns.Name)
	image2, err := build.Build(helper.GetCTX(t), "./testdata/nginx2/acorn.cue", &build.Options{
		Client:    helper.BuilderClient(t),
		Cwd:       "./testdata/nginx2",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, image2.ID, "foo:latest")
	if err != nil {
		t.Fatal(err)
	}

	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	for _, image := range images {
		if image.Name == imageID {
			assert.Equal(t, "", image.Tag)
			assert.Equal(t, "", image.Repository)
		} else if image.Name == image2.ID {
			assert.Equal(t, "latest", image.Tag)
			assert.Equal(t, "foo", image.Repository)
			assert.Equal(t, "foo:latest", image.Reference)
		} else {
			t.Fatal(err, "invalid image")
		}
	}
}

func TestImageTag(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = newImage(t, ns.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	err = c.ImageTag(ctx, image.Name, "foo")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "foo", "bar")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "foo", "ghcr.io/acorn-io/acorn/test:v0.0.0-abc")
	if err != nil {
		t.Fatal(err)
	}

	err = c.ImageTag(ctx, "ghcr.io/acorn-io/acorn/test:v0.0.0-abc", "ghcr.io/acorn-io/acorn/test:v0.0.0-def")
	if err != nil {
		t.Fatal(err)
	}

	newImage, err := c.ImageGet(ctx, "ghcr.io/acorn-io/acorn/test:v0.0.0-abc")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, image.Name, newImage.Name)
	assert.Equal(t, "ghcr.io/acorn-io/acorn/test:v0.0.0-abc", newImage.Reference)
	assert.Equal(t, "ghcr.io/acorn-io/acorn/test", newImage.Repository)
	assert.Equal(t, "v0.0.0-abc", newImage.Tag)
}

func TestImagePush(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = newImage(t, ns.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]
	tagName := registry + "/test:ci"

	err = c.ImageTag(ctx, image.Name, tagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}
}

func TestImagePull(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	id := newImage(t, ns.Name)
	tagName := registry + "/test:ci"

	err = c.ImageTag(ctx, id, tagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	ns = helper.TempNamespace(t, kclient)

	c, err = client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	progress, err = c.ImagePull(ctx, tagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]
	assert.Equal(t, tagName, image.Reference)
}

func TestImageDetails(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	id := newImage(t, ns.Name)
	remoteTagName := registry + "/test:ci"

	err = c.ImageTag(ctx, id, remoteTagName)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	for update := range progress {
		if update.Error != "" {
			t.Fatal(update.Error)
		}
	}

	ns = helper.TempNamespace(t, kclient)

	c, err = client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := newImage(t, ns.Name)

	err = c.ImageTag(ctx, imageID, "foo")
	if err != nil {
		t.Fatal(err)
	}

	details, err := c.ImageDetails(ctx, imageID[:3], nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	_, err = c.ImageDetails(ctx, "a12", nil)
	assert.True(t, apierrors.IsNotFound(err))

	details, err = c.ImageDetails(ctx, "foo:latest", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	details, err = c.ImageDetails(ctx, remoteTagName, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))
}
