package info_test

import (
	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDefaultClientInfoOneNamespace(t *testing.T) {
	helper.StartController(t)
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.NamedTempProject(t, kclient, "test1-project1")

	c, err := client.New(restConfig, ns.Name, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	var infoResponse []v1.Info
	infoResponse, err = c.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, infoResponse, 1, "Default Client's info returned more than 1 response.")
}

func TestDefaultClientInfoTwoNamespace(t *testing.T) {
	helper.StartController(t)
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)

	// Create two projects
	ns1Spec := helper.NamedTempProject(t, kclient, "test2-project1")
	_ = helper.NamedTempProject(t, kclient, "test2-project2")

	// create instance of default-client for a single namespace
	c, err := client.New(restConfig, ns1Spec.Name, ns1Spec.Namespace)
	if err != nil {
		t.Fatal(err)
	}

	var infoResponse []v1.Info
	infoResponse, err = c.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, infoResponse, 1, "Default Client's info returned more than 1 response.")
}

func TestMultiClientInfoThreeNamespace(t *testing.T) {
	helper.StartController(t)
	helper.EnsureCRDs(t)

	ctx := helper.GetCTX(t)

	cliConfig, err := config.ReadCLIConfig()
	if err != nil {
		t.Fatal(err)
	}

	// interface directly with k8 client to create projects
	kclient := helper.MustReturn(kclient.Default)
	helper.NamedTempProject(t, kclient, "test3-project1")
	helper.NamedTempProject(t, kclient, "test3-project2")
	time.Sleep(time.Millisecond * 100)

	// Create multiclient to test commands off of
	mc, err := project.Client(ctx, project.Options{AllProjects: true, CLIConfig: cliConfig})
	if err != nil {
		t.Fatal(err)
	}

	infos, err := mc.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// This will not pass if the local k8 cluster has any additional namespaces.
	assert.Lenf(t, infos, 3, "Multiclient didn't find 3 info responses, found %i.", len(infos))
	expectedProjects := []string{"test3-project1", "test3-project2", "acorn"}
	for _, subInfo := range infos {
		assert.Contains(t, expectedProjects, subInfo.ObjectMeta.Namespace)
	}
}
