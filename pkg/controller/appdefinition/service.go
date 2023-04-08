package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/services"
	"github.com/acorn-io/baaah/pkg/router"
)

func addServices(req router.Request, app *v1.AppInstance, resp router.Response) error {
	objs, err := services.ToAcornServices(req.Ctx, req.Client, app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	return nil
}
