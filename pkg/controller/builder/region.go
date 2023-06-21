package builder

import (
	"github.com/acorn-io/baaah/pkg/router"
	internalv1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/controller/defaults"
)

func SetRegion(req router.Request, _ router.Response) error {
	return defaults.AddDefaultRegion(req.Ctx, req.Client, req.Object.(*internalv1.BuilderInstance))
}
