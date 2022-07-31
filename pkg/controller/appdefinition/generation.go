package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
)

func UpdateGeneration(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	app.Status.ObservedGeneration = app.Generation
	return nil
}
