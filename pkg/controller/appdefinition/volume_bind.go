package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
)

func ReleaseVolume(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	for _, bind := range appInstance.Spec.Volumes {
		pv := &corev1.PersistentVolume{}
		if err := req.Client.Get(pv, bind.Volume, nil); err != nil {
			return err
		}
		if pv.Labels[labels.AcornManaged] == "true" &&
			pv.Status.Phase == corev1.VolumeReleased &&
			pv.Spec.ClaimRef != nil {
			pv.Spec.ClaimRef = nil
			return req.Client.Update(pv)
		}
	}
	return nil
}
