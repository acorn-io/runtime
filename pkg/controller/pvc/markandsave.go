package pvc

import (
	"fmt"

	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func MarkAndSave(req router.Request, resp router.Response) error {
	pvc := req.Object.(*corev1.PersistentVolumeClaim)
	if pvc.Spec.VolumeName == "" {
		return nil
	}

	var pv corev1.PersistentVolume
	if err := req.Client.Get(req.Ctx, kclient.ObjectKey{Name: pvc.Spec.VolumeName}, &pv); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("looking up pv %s", pvc.Spec.VolumeName)
	}

	if pv.Labels[labels.AcornAppName] != pvc.Labels[labels.AcornAppName] ||
		pv.Labels[labels.AcornAppNamespace] != pvc.Labels[labels.AcornAppNamespace] ||
		pv.Labels[labels.AcornVolumeName] != pvc.Name ||
		pv.Labels[labels.AcornVolumeClass] != pvc.Labels[labels.AcornVolumeClass] ||
		pv.Labels[labels.AcornManaged] != "true" ||
		pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		if pv.Labels == nil {
			pv.Labels = map[string]string{}
		}

		pv.Labels[labels.AcornVolumeName] = pvc.Name
		pv.Labels[labels.AcornVolumeClass] = pvc.Labels[labels.AcornVolumeClass]
		pv.Labels[labels.AcornAppName] = pvc.Labels[labels.AcornAppName]
		pv.Labels[labels.AcornAppNamespace] = pvc.Labels[labels.AcornAppNamespace]
		pv.Labels[labels.AcornManaged] = "true"
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		return req.Client.Update(req.Ctx, &pv)
	}

	return nil
}
