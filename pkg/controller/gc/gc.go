package gc

import (
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
)

func GCOrphans(req router.Request, resp router.Response) error {
	if !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}
	return apply.New(req.Client).PurgeOrphan(req.Ctx, req.Object)
}
