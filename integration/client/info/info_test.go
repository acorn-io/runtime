package info_test

import (
	"testing"
	"time"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/config"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/project"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClientInfoOneNamespace(t *testing.T) {
	helper.StartController(t)
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	ns := helper.TempProject(t, kclient)

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
	ns1 := helper.TempProject(t, kclient)
	ns2 := helper.TempProject(t, kclient)
	time.Sleep(time.Millisecond * 100)

	// Create multiclient to test commands off of
	mc, err := project.Client(ctx, project.Options{AllProjects: true, CLIConfig: cliConfig})
	if err != nil {
		t.Fatal(err)
	}

	projectNames := []string{ns1.Name, ns2.Name}

	infos, err := mc.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// check that projectNames is a subset of Info's projects
	subset := helper.Subset[string, v1.Info, string](t, projectNames, infos, func(ele string) string { return ele }, func(info v1.Info) string { return info.Namespace })
	assert.Truef(t, subset, "%+v is not a subset of %+v", projectNames, infos)
}
