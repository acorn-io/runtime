package client

import (
	"strings"
	"testing"

	"github.com/acorn-io/acorn/integration/helper"
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/build"
	hclient "github.com/acorn-io/acorn/pkg/client"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/baaah/pkg/restconfig"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFriendlyNameInContainer(t *testing.T) {
	helper.StartController(t)

	ctx := helper.GetCTX(t)
	client := helper.MustReturn(kclient.Default)
	ns := helper.TempNamespace(t, client)

	image, err := build.Build(helper.GetCTX(t), "./testdata/nginx/acorn.cue", &build.Options{
		Cwd:       "./testdata/nginx",
		Namespace: ns.Name,
	})
	if err != nil {
		t.Fatal(err)
	}

	appInstance := &v1.AppInstance{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "simple-app",
			Namespace:    ns.Name,
		},
		Spec: v1.AppInstanceSpec{
			Image: image.ID,
		},
	}

	err = client.Create(ctx, appInstance)
	if err != nil {
		t.Fatal(err)
	}

	appInstance = helper.WaitForObject(t, client.Watch, &v1.AppInstanceList{}, appInstance, func(obj *v1.AppInstance) bool {
		return obj.Status.ContainerStatus["default"].Ready == 1
	})

	cfg, err := restconfig.Default()
	if err != nil {
		t.Fatal(err)
	}

	c, err := hclient.New(cfg, ns.Name)
	if err != nil {
		t.Fatal(err)
	}

	cs, err := c.ContainerReplicaList(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, img, _ := strings.Cut(image.ImageData.Containers["default"].Image, "@")
	assert.Len(t, cs, 1)
	assert.Equal(t, "nginx", cs[0].Image)
	assert.True(t, strings.HasSuffix(cs[0].Status.ImageID, img))
}
