package run

import (
	"testing"

	"github.com/ibuildthecloud/herd/integration/helper"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/build"
	"github.com/ibuildthecloud/herd/pkg/client"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSimple(t *testing.T) {
	helper.StartController(t)

	image, err := build.Build(helper.GetCTX(t), "./testdata/simple/herd.cue", &build.Opts{
		Cwd: "./testdata/simple",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := helper.GetCTX(t)
	client := client.MustDefault()
	ns := helper.TempNamespace(t, client)
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
		return obj.Status.Conditions[v1.AppInstanceConditionParsed].Success
	})
	assert.NotEmpty(t, appInstance.Status.Namespace)
}
