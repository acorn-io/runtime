package autoupgrade

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
)

func AutoUpgradeOn(next router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		if req.Object == nil {
			return nil
		}

		app := req.Object.(*v1.AppInstance)
		if _, on := Mode(app.Spec); on {
			return next.Handle(req, resp)
		}
		return nil
	})
}
