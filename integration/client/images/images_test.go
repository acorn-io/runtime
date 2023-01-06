package images

import (
	"fmt"
	"strings"
	"testing"

	client2 "github.com/acorn-io/acorn/integration/client"
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
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

	delImage, err := c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: false})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, newImage.Name, delImage.Name)

	_, err = c.ImageGet(ctx, image.Name)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestImageTagMove(t *testing.T) {
	c, ns := helper.ClientAndNamespace(t)
	ctx := helper.GetCTX(t)

	imageID := client2.NewImage(t, ns.Name)
	image2, err := c.AcornImageBuild(ctx, "../testdata/nginx2/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/nginx2",
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
			assert.Equal(t, []string([]string(nil)), image.Tags)
		} else if image.Name == image2.ID {
			assert.Equal(t, "foo:latest", image.Tags[0])
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = client2.NewImage(t, ns.Name)
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
	assert.Equal(t, "ghcr.io/acorn-io/acorn/test:v0.0.0-abc", newImage.Tags[2])
}

func TestImagePush(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	_ = client2.NewImage(t, ns.Name)
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

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	id := client2.NewImage(t, ns.Name)
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

	c, err = client.New(restConfig, "", ns.Name)
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
	assert.Equal(t, tagName, image.Tags[0])
}

func TestImageDetails(t *testing.T) {
	helper.StartController(t)
	registry, close := helper.StartRegistry(t)
	defer close()
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	id := client2.NewImage(t, ns.Name)
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

	c, err = client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)

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

func TestImageDeleteTwoTags(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	err = c.ImageTag(ctx, image.Name, "repo:tag1")
	if err != nil {
		t.Fatal(err)
	}
	err = c.ImageTag(ctx, image.Name, "repo:tag2")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: false})
	expectedErr := fmt.Errorf("unable to delete %s (must be forced) - image is referenced in multiple repositories", image.Name)
	assert.Equal(t, expectedErr, err)

	_, err = c.ImageDelete(ctx, image.Name, &client.ImageDeleteOptions{Force: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.ImageGet(ctx, image.Name)
	assert.True(t, apierrors.IsNotFound(err))
}

func TestImageBadTag(t *testing.T) {
	helper.StartController(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, kclient)

	c, err := client.New(restConfig, "", ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	imageID := client2.NewImage(t, ns.Name)
	images, err := c.ImageList(ctx)
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, images, 1)

	image := images[0]

	assert.Equal(t, imageID, image.Name)
	assert.False(t, strings.HasPrefix(imageID, "sha256:"))
	assert.Equal(t, "sha256:"+image.Name, image.Digest)

	err = c.ImageTag(ctx, image.Name, "foo:a@badtag")
	assert.Equal(t, "tag can only contain the characters `abcdefghijklmnopqrstuvwxyz0123456789_-.ABCDEFGHIJKLMNOPQRSTUVWXYZ`: a@badtag", err.Error())

	err = c.ImageTag(ctx, image.Name, "foo@@:badtag")
	assert.Equal(t, "repository can only contain the characters `abcdefghijklmnopqrstuvwxyz0123456789_-./`: foo@@", err.Error())

}
