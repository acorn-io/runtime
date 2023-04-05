package appdefinition

import (
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
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

func TranslateVolume(req router.Request, resp router.Response) error {
	pv := req.Object.(*corev1.PersistentVolume)
	i := strings.LastIndex(pv.Name, ".")
	if i == -1 || i+1 >= len(pv.Name) {
		return nil
	}

	// parse it of the form <appName>.<shortVolName>
	prefix := pv.Name[:i]
	volumeName := pv.Name[i+1:]

	var acornPV corev1.PersistentVolumeList
	err := req.Client.List(req.Ctx, &acornPV, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornAppName:    prefix,
			labels.AcornVolumeName: volumeName,
		}),
	})
	if err == nil && len(acornPV.Items) == 1 {
		// replace alias name with PVC name
		fmt.Printf("gah")
	}

	return nil
}

func ParseVolumeBindingsMiddleware(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		status := condition.Setter(appInstance, resp, v1.AppInstanceConditionParsed)
		appVolumeBindings := appInstance.Spec.Volumes

		if len(appVolumeBindings) == 0 {
			return h.Handle(req, resp)
		}

		var acornPV corev1.PersistentVolumeList
		for index, vol := range appVolumeBindings {
			i := strings.LastIndex(vol.Volume, ".")
			if i == -1 || i+1 >= len(vol.Volume) {
				continue
			}

			// parse it of the form <appName>.<shortVolName>
			prefix := vol.Volume[:i]
			volumeName := vol.Volume[i+1:]

			err := req.Client.List(req.Ctx, &acornPV, &kclient.ListOptions{
				LabelSelector: klabels.SelectorFromSet(map[string]string{
					labels.AcornAppName:    prefix,
					labels.AcornVolumeName: volumeName,
				}),
			})
			if err == nil && len(acornPV.Items) == 1 {
				// replace alias name with PVC name
				appInstance.Spec.Volumes[index].Volume = acornPV.Items[0].Name
			}
		}

		req.Object = appInstance
		status.Success()

		return h.Handle(req, resp)
	})
}
