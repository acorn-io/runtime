package client

import (
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
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
	assert.Equal(t, []string{"foo", "bar",
		"ghcr.io/acorn-io/acorn/test:v0.0.0-abc",
		"ghcr.io/acorn-io/acorn/test:v0.0.0-def",
	}, newImage.Tags)
}

func TestImagePush(t *testing.T) {
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

	err = c.ImageTag(ctx, image.Name, "ghcr.io/acorn-io/acorn/test:ci")
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePush(ctx, "ghcr.io/acorn-io/acorn/test:ci", nil)
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
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	progress, err := c.ImagePull(ctx, "ghcr.io/acorn-io/acorn/test:ci", nil)
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
	assert.Equal(t, "ghcr.io/acorn-io/acorn/test:ci", image.Tags[0])
}

func TestImageDetails(t *testing.T) {
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

	details, err := c.ImageDetails(ctx, imageID, nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))

	details, err = c.ImageDetails(ctx, "docker.io/ibuildthecloud/test:ci", nil)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, strings.Contains(details.AppImage.Acornfile, "nginx"))
}
