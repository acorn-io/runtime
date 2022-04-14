package pvc

import (
	"fmt"

	"github.com/ibuildthecloud/baaah/pkg/router"
	"github.com/ibuildthecloud/herd/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func MarkAndSave(req router.Request, resp router.Response) error {
	pvc := req.Object.(*corev1.PersistentVolumeClaim)
	if pvc.Spec.VolumeName == "" {
		return nil
	}

	var pv corev1.PersistentVolume
	if err := req.Client.Get(&pv, pvc.Spec.VolumeName, nil); apierrors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("looking up pv %s", pvc.Spec.VolumeName)
	}

	if pv.Labels[labels.HerdAppName] != pvc.Labels[labels.HerdAppName] ||
		pv.Labels[labels.HerdAppNamespace] != pvc.Labels[labels.HerdAppNamespace] ||
		pv.Labels[labels.HerdVolumeName] != pvc.Name ||
		pv.Labels[labels.HerdManaged] != "true" ||
		pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		if pv.Labels == nil {
			pv.Labels = map[string]string{}
		}

		pv.Labels[labels.HerdVolumeName] = pvc.Name
		pv.Labels[labels.HerdAppName] = pvc.Labels[labels.HerdAppName]
		pv.Labels[labels.HerdAppNamespace] = pvc.Labels[labels.HerdAppNamespace]
		pv.Labels[labels.HerdManaged] = "true"
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		return req.Client.Update(&pv)
	}

	return nil
}
