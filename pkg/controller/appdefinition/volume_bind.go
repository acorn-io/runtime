package appdefinition

import (
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
)

func ReleaseVolume(req router.Request, resp router.Response) error {
	pv := req.Object.(*corev1.PersistentVolume)
	if pv.Labels[labels.AcornManaged] == "true" &&
		pv.Status.Phase == corev1.VolumeReleased &&
		pv.Spec.ClaimRef != nil {
		pv.Spec.ClaimRef = nil
		return req.Client.Update(req.Ctx, pv)
	}
	return nil
}
