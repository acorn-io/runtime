package gc

import (
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
)

func Orphans(req router.Request, _ router.Response) error {
	if !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}

	// Handle migration to ServiceInstance for Builders
	svc, ok := req.Object.(*corev1.Service)
	if ok && svc.Namespace == system.ImagesNamespace && svc.Annotations[apply.LabelGVK] == "internal.acorn.io/v1, Kind=BuilderInstance" {
		return req.Client.Delete(req.Ctx, svc)
	}

	return apply.New(req.Client).PurgeOrphan(req.Ctx, req.Object)
}
