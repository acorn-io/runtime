package appdefinition

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/secrets"
	"github.com/acorn-io/acorn/pkg/volume"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/baaah/pkg/uncached"
	name2 "github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AcornHelper     = " /acorn-helper"
	AcornHelperPath = "/.acorn"
)

func addPVCs(req router.Request, appInstance *v1.AppInstance, resp router.Response) error {
	pvcs, err := toPVCs(req, appInstance)
	if err != nil {
		return err
	}
	resp.Objects(pvcs...)
	return nil
}

func translateAccessModes(accessModes []v1.AccessMode) []corev1.PersistentVolumeAccessMode {
	if len(accessModes) == 0 {
		return []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}

	result := make([]corev1.PersistentVolumeAccessMode, 0, len(accessModes))
	for _, accessMode := range accessModes {
		newMode := strings.ToUpper(string(accessMode[0:1])) + string(accessMode[1:])
		result = append(result, corev1.PersistentVolumeAccessMode(newMode))
	}
	return result
}

func lookupExistingPV(req router.Request, appInstance *v1.AppInstance, volumeName string) (string, error) {
	var pvc corev1.PersistentVolumeClaim
	if err := req.Get(&pvc, appInstance.Status.Namespace, volumeName); err == nil {
		return pvc.Spec.VolumeName, nil
	} else if !apierrors.IsNotFound(err) {
		return "", err
	}

	if err := req.Get(uncached.Get(&pvc), appInstance.Status.Namespace, volumeName); err == nil {
		return pvc.Spec.VolumeName, nil
	} else if !apierrors.IsNotFound(err) {
		return "", err
	}

	var pv corev1.PersistentVolumeList
	err := req.List(uncached.List(&pv), &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged:      "true",
			labels.AcornAppName:      appInstance.Name,
			labels.AcornAppNamespace: appInstance.Namespace,
			labels.AcornVolumeName:   volumeName,
		}),
	})
	if err != nil {
		return "", err
	}

	switch len(pv.Items) {
	case 0:
		return "", nil
	case 1:
		return pv.Items[0].Name, nil
	default:
		names := typed.MapSlice(pv.Items, func(pv corev1.PersistentVolume) string {
			return pv.Name
		})
		return "", fmt.Errorf("can not bind existing volume, there are more that one valid volumes %v", names)
	}
}

func toPVCs(req router.Request, appInstance *v1.AppInstance) (result []kclient.Object, err error) {
	volumeClasses, _, err := volume.GetVolumeClasses(req.Ctx, req.Client, appInstance.Namespace)
	if err != nil {
		return nil, err
	}

	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Volumes) {
		vol, volumeRequest := entry.Key, entry.Value

		var volumeBinding, bind = isBind(appInstance, vol)

		if volumeRequest.Class == v1.VolumeRequestTypeEphemeral && !bind {
			continue
		}

		volumeRequest = volume.CopyVolumeDefaults(volumeRequest, volumeBinding, appInstance.Status.Defaults.Volumes[vol])

		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vol,
				Namespace: appInstance.Status.Namespace,
				Labels:    volumeLabels(appInstance, vol, volumeRequest),
				Annotations: labels.GatherScoped(vol, v1.LabelTypeVolume, appInstance.Status.AppSpec.Annotations,
					volumeRequest.Annotations, appInstance.Spec.Annotations),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: translateAccessModes(volumeRequest.AccessModes),
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{},
				},
			},
		}

		if appInstance.Generation > 0 {
			pvc.Annotations[labels.AcornAppGeneration] = strconv.FormatInt(appInstance.Generation, 10)
		}

		if bind {
			pvc.Name = bindName(vol)
			pvc.Spec.VolumeName = volumeBinding.Volume
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MinSize

			if volumeBinding.Class != "" {
				// Specifically allowing volume classes that are inactive.
				if volClass, ok := volumeClasses[volumeBinding.Class]; !ok {
					return nil, fmt.Errorf("%s has an invalid volume class %s", vol, volumeBinding.Class)
				} else {
					pvc.Spec.StorageClassName = &volClass.StorageClassName
					pvc.Labels[labels.AcornVolumeClass] = volClass.Name
				}
			} else {
				// User did not specify a class with the binding, so get the class from the existing volume.
				pv := new(corev1.PersistentVolume)
				if err = req.Get(pv, "", volumeBinding.Volume); err != nil {
					return nil, err
				}
				pvc.Labels[labels.AcornVolumeClass] = pv.Labels[labels.AcornVolumeClass]
			}

			if volumeBinding.Size != "" {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MustParseResourceQuantity(volumeBinding.Size)
			}
		} else {
			if volumeRequest.Class != "" {
				// Specifically allowing volume classes that are inactive.
				if volClass, ok := volumeClasses[volumeRequest.Class]; !ok && volumeBinding.Class == "" {
					return nil, fmt.Errorf("%s has an invalid volume class %s", vol, volumeRequest.Class)
				} else {
					pvc.Spec.StorageClassName = &volClass.StorageClassName
					pvc.Labels[labels.AcornVolumeClass] = volClass.Name
				}
			}
			pvName, err := lookupExistingPV(req, appInstance, vol)
			if err != nil {
				return nil, err
			}
			pvc.Spec.VolumeName = pvName

			if volumeRequest.Size == "" {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.DefaultSize
			} else {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MustParseResourceQuantity(volumeRequest.Size)
			}
		}

		result = append(result, &pvc)
	}
	return
}

func volumeLabels(appInstance *v1.AppInstance, volume string, volumeRequest v1.VolumeRequest) map[string]string {
	labelMap := map[string]string{
		labels.AcornAppName:      appInstance.Name,
		labels.AcornAppNamespace: appInstance.Namespace,
		labels.AcornManaged:      "true",
		labels.AcornVolumeName:   volume,
	}
	return labels.Merge(labelMap, labels.GatherScoped(volume, v1.LabelTypeVolume, appInstance.Status.AppSpec.Labels,
		volumeRequest.Labels, appInstance.Spec.Labels))
}

func isEphemeral(appInstance *v1.AppInstance, volume string) (v1.VolumeRequest, bool) {
	if volume == AcornHelper && appInstance.Spec.GetDevMode() {
		return v1.VolumeRequest{
			Class: v1.VolumeRequestTypeEphemeral,
		}, true
	}
	for name, volumeRequest := range appInstance.Status.AppSpec.Volumes {
		if name == volume && strings.EqualFold(volumeRequest.Class, v1.VolumeRequestTypeEphemeral) {
			return volumeRequest, true
		}
	}
	return v1.VolumeRequest{}, false
}

func isBind(appInstance *v1.AppInstance, volume string) (v1.VolumeBinding, bool) {
	for _, v := range appInstance.Spec.Volumes {
		if v.Target == volume {
			return v, v.Volume != ""
		}
	}
	return v1.VolumeBinding{}, false
}

func bindName(volume string) string {
	return name2.SafeConcatName(volume, "bind")
}

func toVolumeName(appInstance *v1.AppInstance, volume string) (string, bool) {
	if _, bind := isBind(appInstance, volume); bind {
		return bindName(volume), true
	}
	return volume, false
}

func addVolumeReferencesForContainer(app *v1.AppInstance, volumeReferences map[volumeReference]bool, container v1.Container) {
	for _, entry := range typed.Sorted(container.Dirs) {
		volume := entry.Value
		if volume.ContextDir != "" {
			if app.Spec.GetDevMode() {
				volumeReferences[volumeReference{name: AcornHelper}] = true
			}
		} else if volume.Secret.Name == "" {
			volumeReferences[volumeReference{name: volume.Volume}] = true
		} else {
			volumeReferences[volumeReference{secretName: volume.Secret.Name}] = true
		}
	}

	for _, entry := range typed.Sorted(container.Files) {
		file := entry.Value
		if file.Secret.Name != "" {
			volumeReferences[volumeReference{secretName: file.Secret.Name, mode: file.Mode}] = true
		}
	}
}

type volumeReference struct {
	name       string
	secretName string
	mode       string
}

func (v volumeReference) Suffix() string {
	if volume.NormalizeMode(v.mode) == "" {
		return ""
	}
	return "-" + v.mode
}

func toMode(m string) (*int32, error) {
	i, err := strconv.ParseInt(m, 8, 32)
	if err != nil {
		return nil, err
	}
	i32 := int32(i)
	return &i32, nil
}

func (v volumeReference) ParseMode() (*int32, error) {
	if volume.NormalizeMode(v.mode) == "" {
		return nil, nil
	}
	return toMode(v.mode)
}

func addFilesFileModesForContainer(fileModes map[string]bool, container v1.Container) {
	for _, file := range container.Files {
		if file.Content != "" && file.Secret.Name == "" {
			fileModes[volume.NormalizeMode(file.Mode)] = true
		}
	}
	for _, sidecar := range container.Sidecars {
		for _, file := range sidecar.Files {
			if file.Content != "" && file.Secret.Name == "" {
				fileModes[volume.NormalizeMode(file.Mode)] = true
			}
		}
	}
}

func toVolumes(appInstance *v1.AppInstance, container v1.Container, interpolator *secrets.Interpolator) (result []corev1.Volume, _ error) {
	volumeReferences := map[volumeReference]bool{}
	addVolumeReferencesForContainer(appInstance, volumeReferences, container)
	for _, entry := range typed.Sorted(container.Sidecars) {
		addVolumeReferencesForContainer(appInstance, volumeReferences, entry.Value)
	}

	for volume := range volumeReferences {
		if volume.secretName != "" {
			mode, err := volume.ParseMode()
			if err != nil {
				return nil, err
			}
			result = append(result, corev1.Volume{
				Name: "secret--" + volume.secretName + volume.Suffix(),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  volume.secretName,
						DefaultMode: mode,
					},
				},
			})
			continue
		}

		name, bind := toVolumeName(appInstance, volume.name)
		if vr, ok := isEphemeral(appInstance, volume.name); ok && !bind {
			result = append(result, corev1.Volume{
				Name: sanitizeVolumeName(volume.name),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: v1.MustParseResourceQuantity(vr.Size),
					},
				},
			})
		} else {
			result = append(result, corev1.Volume{
				Name: volume.name,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: name,
					},
				},
			})
		}
	}

	fileModes := map[string]bool{}
	addFilesFileModesForContainer(fileModes, container)

	secretName := interpolator.SecretName()
	for _, modeString := range typed.SortedKeys(fileModes) {
		var (
			mode *int32
			err  error
			name = interpolator.SecretName()
		)
		if modeString != "" {
			name = name + "-" + modeString
			mode, err = toMode(modeString)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: mode,
					SecretName:  secretName,
				},
			},
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return
}
