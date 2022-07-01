package gc

import (
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
)

func GCOrphans(req router.Request, resp router.Response) error {
	return apply.New(req.Client).PurgeOrphan(req.Ctx, req.Object)
}
