package info_test

import (
	"testing"
	"time"

	"github.com/acorn-io/runtime/integration/helper"
	v1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/client"
	kclient "github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/project"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClientInfoOneNamespace(t *testing.T) {
	helper.StartController(t)
	helper.EnsureCRDs(t)
	restConfig := helper.StartAPI(t)

	ctx := helper.GetCTX(t)
	kclient := helper.MustReturn(kclient.Default)
	project := helper.TempProject(t, kclient)

	c, err := client.New(restConfig, project.Name, project.Name)
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
	project1 := helper.TempProject(t, kclient)
	_ = helper.TempProject(t, kclient)

	// create instance of default-client for a single namespace
	c, err := client.New(restConfig, project1.Name, project1.Namespace)
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

	// interface directly with k8 client to create projects
	kclient := helper.MustReturn(kclient.Default)
	project1 := helper.TempProject(t, kclient)
	project2 := helper.TempProject(t, kclient)
	time.Sleep(time.Millisecond * 100)

	// Create multiclient to test commands off of
	mc, err := project.Client(ctx, project.Options{
		AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}

	projectNames := []string{project1.Name, project2.Name}

	infos, err := mc.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// check that projectNames is a subset of Info's projects
	subset := helper.Subset[string, v1.Info, string](t, projectNames, infos, func(ele string) string {
		return ele
	}, func(info v1.Info) string {
		return info.Namespace
	})
	assert.Truef(t, subset, "%+v is not a subset of %+v", projectNames, infos)
}
