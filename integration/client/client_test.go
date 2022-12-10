package client

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	hclient "github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func TestAppsSSA(t *testing.T) {
	cfg := helper.StartAPI(t)
	c, ns := helper.ClientAndNamespace(t)
	ctx := helper.GetCTX(t)

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(ctx, "./testdata/nginx/Acornfile", &hclient.AcornImageBuildOptions{
		Cwd: "./testdata/nginx",
	})
	if err != nil {
		t.Fatal(err)
	}

	appBytes, err := json.Marshal(&apiv1.App{
		TypeMeta: metav1.TypeMeta{
			Kind:       "App",
			APIVersion: api.Group + "/" + apiv1.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = dyn.Resource(apiv1.SchemeGroupVersion.WithResource("apps")).
		Namespace(ns.Name).
		Patch(ctx, "test", types.ApplyPatchType, appBytes, metav1.PatchOptions{
			FieldManager: "unit-test",
		})
	if err != nil {
		t.Fatal(err)
	}

	app, err := c.AppGet(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, image.ID, app.Spec.Image)
}

func TestFriendlyNameInContainer(t *testing.T) {
	helper.StartController(t)
	cfg := helper.StartAPI(t)

	c, ns := helper.ClientAndNamespace(t)
	ctx := helper.GetCTX(t)
	client, err := k8sclient.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	image, err := c.AcornImageBuild(helper.GetCTX(t), "./testdata/nginx/Acornfile", &hclient.AcornImageBuildOptions{
		Cwd: "./testdata/nginx",
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &apiv1.App{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
		},
	}

	err = client.Create(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	helper.WaitForObject(t, client.Watch, &apiv1.AppList{}, appInstance, func(obj *apiv1.App) bool {
		return obj.Status.ContainerStatus["default"].Ready == 1
	})

	c, err = hclient.New(cfg, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cs, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, img, _ := strings.Cut(image.ImageData.Containers["default"].Image, "@")
	assert.Len(t, cs, 1)
	assert.Equal(t, "public.ecr.aws/docker/library/nginx:latest", cs[0].Spec.Image)
	assert.True(t, strings.HasSuffix(cs[0].Status.ImageID, img))
}
