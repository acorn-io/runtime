package appstatus

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/volume"
	"golang.org/x/exp/maps"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *appStatusRenderer) readVolumes() error {
	// reset
	a.app.Status.AppStatus.Volumes = make(map[string]v1.VolumeStatus, len(a.app.Status.AppSpec.Volumes))

	for volumeName, vol := range a.app.Status.AppSpec.Volumes {
		isEphemeral := vol.Class == v1.VolumeRequestTypeEphemeral
		a.app.Status.AppStatus.Volumes[volumeName] = v1.VolumeStatus{
			CommonStatus: v1.CommonStatus{
				LinkOverride: linkedVolume(a.app, volumeName),
				Defined:      isEphemeral,
				UpToDate:     isEphemeral,
				Ready:        isEphemeral,
			},
			Bound:             isEphemeral,
			StorageClassFound: isEphemeral,
		}
	}

	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := a.c.List(a.ctx, pvcs, &kclient.ListOptions{
		Namespace: a.app.Status.Namespace,
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
			labels.AcornAppName: a.app.Name,
		}),
	}); err != nil {
		return err
	} else if len(pvcs.Items) == 0 {
		return nil
	}

	storageClassNames, err := volume.GetVolumeClassNames(a.ctx, a.c, a.app.Namespace, true)
	if err != nil {
		return err
	}

	sort.Slice(pvcs.Items, func(i, j int) bool {
		return pvcs.Items[i].CreationTimestamp.Before(&pvcs.Items[j].CreationTimestamp)
	})

	for _, pvc := range pvcs.Items {
		volumeName := pvc.Labels[labels.AcornVolumeName]
		if volumeName == "" {
			continue
		}

		v := a.app.Status.AppStatus.Volumes[volumeName]
		v.Defined = true
		v.UpToDate = pvc.Annotations[labels.AcornAppGeneration] == strconv.Itoa(int(a.app.Generation))

		if pvc.Spec.VolumeName != "" {
			pv := &corev1.PersistentVolume{}
			if err := a.c.Get(a.ctx, router.Key("", pvc.Spec.VolumeName), pv); apierrors.IsNotFound(err) || err == nil {
				v.VolumeName = publicname.Get(pv)
			} else {
				return err
			}
		}

		switch pvc.Status.Phase {
		case corev1.ClaimBound:
			// No message if the PVC is in phase bound.
			v.Ready = true
			v.Bound = true
			v.StorageClassFound = true
		default:
			// ignore volumes not mounted because they will never be bound
			if a.volumeIsUsed(volumeName) {
				if pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName != "" && !slices.Contains(storageClassNames, *pvc.Spec.StorageClassName) {
					v.ErrorMessages = append(v.ErrorMessages, fmt.Sprintf("volume class %s for volume %s doesn't exist", *pvc.Spec.StorageClassName, pvc.Labels[labels.AcornVolumeName]))
				} else {
					v.StorageClassFound = true
				}
				v.TransitioningMessages = append(v.TransitioningMessages, fmt.Sprintf("waiting for volume %s to provision and bind", pvc.Labels[labels.AcornVolumeName]))
			} else {
				v.Unused = true
				v.Ready = true
			}
		}

		// Not ready if we have any error messages
		if len(v.ErrorMessages) > 0 {
			v.Ready = false
		}

		if v.Ready {
			if v.Unused {
				v.State = "defined"
			} else {
				v.State = "provisioned"
			}
		} else if v.UpToDate {
			if len(v.ErrorMessages) > 0 {
				v.State = "failing"
			} else {
				v.State = "provisioning"
			}
		} else if v.Defined {
			if len(v.ErrorMessages) > 0 {
				v.State = "error"
			} else {
				v.State = "updating"
			}
		} else {
			if len(v.ErrorMessages) > 0 {
				v.State = "error"
			} else {
				v.State = "pending"
			}
		}

		a.app.Status.AppStatus.Volumes[volumeName] = v
	}

	return nil
}

func linkedVolume(app *v1.AppInstance, name string) string {
	if name == "" {
		return ""
	}

	for _, binding := range app.Spec.Volumes {
		if binding.Target == name {
			return binding.Volume
		}
	}

	return ""
}

func (a *appStatusRenderer) volumeIsUsed(volumeName string) bool {
	for _, container := range append(maps.Values(a.app.Status.AppSpec.Containers), maps.Values(a.app.Status.AppSpec.Jobs)...) {
		for _, mount := range container.Dirs {
			if mount.Volume == volumeName {
				return true
			}
		}
		for _, sidecar := range container.Sidecars {
			for _, mount := range sidecar.Dirs {
				if mount.Volume == volumeName {
					return true
				}
			}
		}
	}
	return false
}
