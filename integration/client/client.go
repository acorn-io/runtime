package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
)

func NewImageWithSidecar(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := NewImageFromPath(t, helper.GetCTX(t), c, "../testdata/sidecar")
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func NewImage2(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := NewImageFromPath(t, helper.GetCTX(t), c, "../testdata/nginx2")
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func NewImage(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := NewImageFromPath(t, helper.GetCTX(t), c, "../testdata/nginx")
	if err != nil {
		t.Fatal(err)
	}

	return image.ID
}

func NewImageFromPath(t *testing.T, ctx context.Context, c client.Client, path string) (*v1.AppImage, error) {
	t.Helper()

	image, err := c.AcornImageBuild(ctx, path+"/Acornfile", &client.AcornImageBuildOptions{
		Cwd: path,
	})
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		_, err1 := c.ImageDelete(ctx, image.ID, &client.ImageDeleteOptions{Force: true})
		_, err2 := c.AcornImageBuildDelete(ctx, image.ID)
		if err1 != nil && err2 != nil {
			fmt.Printf("Issue deleting image %s", image.Name)
		}
	})
	return image, nil
}
