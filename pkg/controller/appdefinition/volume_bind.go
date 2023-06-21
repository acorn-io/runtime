package appdefinition

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func ReleaseVolume(req router.Request, resp router.Response) error {
	pv := req.Object.(*corev1.PersistentVolume)
	if pv.Labels[labels.AcornManaged] == "true" &&
		pv.Status.Phase == corev1.VolumeReleased &&
		pv.Spec.ClaimRef != nil {
		app := &v1.AppInstance{}
		err := req.Get(app, pv.Labels[labels.AcornAppNamespace], pv.Labels[labels.AcornAppName])
		if apierror.IsNotFound(err) {
			return nil
		} else if err != nil {
			return err
		} else if app.Status.Namespace == "" {
			return nil
		}

		pvc := &corev1.PersistentVolumeClaim{}
		err = req.Get(pvc, app.Status.Namespace, pv.Labels[labels.AcornVolumeName])
		if apierror.IsNotFound(err) {
			return nil
		} else if err != nil {
			return err
		}
		if pvc.DeletionTimestamp.IsZero() && pvc.Spec.VolumeName == pv.Name {
			pv.Spec.ClaimRef = nil
			return req.Client.Update(req.Ctx, pv)
		}
	}
	return nil
}
