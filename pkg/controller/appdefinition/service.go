package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/secrets"
	"github.com/acorn-io/runtime/pkg/services"
)

func addServices(req router.Request, app *v1.AppInstance, interpolar *secrets.Interpolator, resp router.Response) error {
	objs, err := services.ToAcornServices(req.Ctx, req.Client, interpolar, app)
	if err != nil {
		return err
	}
	resp.Objects(objs...)
	return nil
}
