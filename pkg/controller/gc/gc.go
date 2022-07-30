package gc

import (
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/uncached"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func GCOrphans(req router.Request, resp router.Response) error {
	if !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}

	pod, ok := req.Object.(*corev1.Pod)
	if ok {
		// Purge old debug shells
		if pod.Labels[labels.AcornDebugShell] == "true" && pod.Status.Phase != corev1.PodRunning &&
			pod.Status.Phase != corev1.PodPending {
			return req.Client.Delete(req.Ctx, pod)
		}

		if len(pod.OwnerReferences) != 1 {
			return nil
		}
		if pod.OwnerReferences[0].Kind != "ReplicaSet" {
			return nil
		}
		var rs appsv1.ReplicaSet
		if err := req.Get(&rs, pod.Namespace, pod.OwnerReferences[0].Name); apierror.IsNotFound(err) {
			if err := req.Get(uncached.Get(&rs), pod.Namespace, pod.OwnerReferences[0].Name); apierror.IsNotFound(err) {
				return req.Client.Delete(req.Ctx, pod)
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return apply.New(req.Client).PurgeOrphan(req.Ctx, req.Object)
}
