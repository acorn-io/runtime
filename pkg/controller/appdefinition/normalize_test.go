package appdefinition

import (
	"testing"

	"github.com/ibuildthecloud/baaah/pkg/router/tester"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/appdefinition"
	"github.com/ibuildthecloud/herd/pkg/scheme"
)

func TestParseAppImage(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/parseappimage", ParseAppImage)
}

func TestParseAppImageBug(t *testing.T) {
	appImage := &v1.AppImage{
		ImageData: v1.ImagesData{
			Containers: map[string]v1.ContainerData{},
		},
		Herdfile: "",
		ID:       "",
	}
	app, err := appdefinition.FromAppImage(appImage)
	if err != nil {
		t.Fatal(err)
	}

	_, err = app.AppSpec()
	if err != nil {
		t.Fatal(err)
	}
}
