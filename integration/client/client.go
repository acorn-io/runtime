package client

import (
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	"github.com/acorn-io/acorn/pkg/build"
)

func NewImage2(t *testing.T, namespace string) string {
	image, err := build.Build(helper.GetCTX(t), "../testdata/nginx2/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, namespace),
		Cwd:    "../testdata/nginx2",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func NewImage(t *testing.T, namespace string) string {
	image, err := build.Build(helper.GetCTX(t), "../testdata/nginx/Acornfile", &build.Options{
		Client: helper.BuilderClient(t, namespace),
		Cwd:    "../testdata/nginx",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}
