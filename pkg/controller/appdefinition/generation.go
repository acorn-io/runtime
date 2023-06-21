package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

func UpdateObservedFields(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.ObservedImageDigest = app.Status.AppImage.Digest
	app.Status.ObservedGeneration = app.Generation
	return nil
}
