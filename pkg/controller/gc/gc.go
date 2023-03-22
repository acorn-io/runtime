package gc

import (
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
)

func GCOrphans(req router.Request, resp router.Response) error {
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
