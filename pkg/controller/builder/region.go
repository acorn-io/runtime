package builder

import (
	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/controller/defaults"
	"github.com/acorn-io/baaah/pkg/router"
)

func SetRegion(req router.Request, _ router.Response) error {
	return defaults.AddDefaultRegion(req.Ctx, req.Client, req.Object.(*internalv1.BuilderInstance))
}
