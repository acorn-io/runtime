package pv

import (
	"github.com/ibuildthecloud/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
)

func ReleaseClaim(req router.Request, resp router.Response) error {
	pv := req.Object.(*corev1.PersistentVolume)
	if pv.Status.Phase != corev1.VolumeReleased {
		return nil
	}
	if pv.Spec.ClaimRef != nil {
		pv.Spec.ClaimRef = nil
		return req.Client.Update(pv)
	}
	return nil
}
