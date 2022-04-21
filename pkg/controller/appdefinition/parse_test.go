package appdefinition

import (
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/appdefinition"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router/tester"
)

func TestParseAppImage(t *testing.T) {
	tester.DefaultTest(t, scheme.Scheme, "testdata/parseappimage", ParseAppImage)
}

func TestParseAppImageBug(t *testing.T) {
	appImage := &v1.AppImage{
		ImageData: v1.ImagesData{
			Containers: map[string]v1.ContainerData{},
		},
		Acornfile: "",
		ID:        "",
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
