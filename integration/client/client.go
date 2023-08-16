package client

import (
	"testing"

	"github.com/acorn-io/runtime/integration/helper"
	"github.com/acorn-io/runtime/pkg/client"
	"github.com/stretchr/testify/require"
)

func NewImageWithSidecar(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "../testdata/sidecar/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/sidecar",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func NewImageWithJobs(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "../testdata/job/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/job",
	})
	require.NoError(t, err)

	return image.ID
}

func NewImage2(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "../testdata/nginx2/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/nginx2",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}

func NewImage(t *testing.T, namespace string) string {
	t.Helper()

	c := helper.BuilderClient(t, namespace)
	image, err := c.AcornImageBuild(helper.GetCTX(t), "../testdata/nginx/Acornfile", &client.AcornImageBuildOptions{
		Cwd: "../testdata/nginx",
	})
	if err != nil {
		t.Fatal(err)
	}
	return image.ID
}
