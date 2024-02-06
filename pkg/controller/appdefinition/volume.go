package appdefinition

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/acorn-io/baaah/pkg/name"
	name2 "github.com/acorn-io/baaah/pkg/name"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"github.com/acorn-io/baaah/pkg/uncached"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/secrets"
	"github.com/acorn-io/runtime/pkg/volume"
	"github.com/acorn-io/z"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AcornHelper            = " /acorn-helper"
	AcornHelperPath        = "/.acorn"
	AcornHelperSleepPath   = "/.acorn/sleep"
	AcornHelperBusyboxPath = "/.acorn/busybox"
)

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

func LookupExistingPV(req router.Request, appInstance *v1.AppInstance, volumeName string) (string, error) {
	var pvc corev1.PersistentVolumeClaim
	if err := req.Get(&pvc, appInstance.Status.Namespace, volumeName); err == nil {
		if pvc.Spec.VolumeName == "" {
			return "", nil
		}
		pv := corev1.PersistentVolume{}
		if err := req.Get(&pv, "", pvc.Spec.VolumeName); err == nil {
			if pv.DeletionTimestamp.IsZero() {
				return pvc.Spec.VolumeName, nil
			}
		} else if !apierrors.IsNotFound(err) {
			return "", err
		}
	} else if !apierrors.IsNotFound(err) {
		return "", err
	}

	// same thing as above but uncached
	if err := req.Get(uncached.Get(&pvc), appInstance.Status.Namespace, volumeName); err == nil {
		if pvc.Spec.VolumeName == "" {
			return "", nil
		}
		pv := corev1.PersistentVolume{}
		if err := req.Get(uncached.Get(&pv), "", pvc.Spec.VolumeName); err == nil {
			if pv.DeletionTimestamp.IsZero() {
				return pvc.Spec.VolumeName, nil
			}
		} else if !apierrors.IsNotFound(err) {
			return "", err
		}
		// at this point we have to delete the PVC so that we can reset the invalid pvc.Spec.VolumeName
		if err := req.Client.Delete(req.Ctx, &pvc); err != nil {
			return "", err
		}
	} else if !apierrors.IsNotFound(err) {
		return "", err
	}

	var pv corev1.PersistentVolumeList
	err := req.List(uncached.List(&pv), &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged:      "true",
			labels.AcornAppNamespace: appInstance.Namespace,
			labels.AcornPublicName:   publicname.ForChild(appInstance, volumeName),
		}),
	})
	if err != nil {
		return "", err
	}

	switch len(pv.Items) {
	case 0:
		return "", nil
	case 1:
		if !isPVAvailable(&pv.Items[0], appInstance.Name) {
			return "", fmt.Errorf("cannot use existing volume %s - it is in phase %s", pv.Items[0].Name, pv.Items[0].Status.Phase)
		}
		return pv.Items[0].Name, nil
	default:
		names := typed.MapSlice(pv.Items, func(pv corev1.PersistentVolume) string {
			return pv.Name
		})
		return "", fmt.Errorf("can not bind existing volume, there are more that one valid volumes %v", names)
	}
}

func toPVCs(req router.Request, appInstance *v1.AppInstance) (result []kclient.Object, err error) {
	volumeClasses, _, err := volume.GetVolumeClassInstances(req.Ctx, req.Client, appInstance.Namespace)
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
				Annotations: typed.Concat(
					labels.GatherScoped(vol, v1.LabelTypeVolume, appInstance.Status.AppSpec.Annotations, volumeRequest.Annotations, appInstance.Spec.Annotations),
					map[string]string{labels.AcornConfigHashAnnotation: appInstance.Status.AppStatus.Volumes[vol].ConfigHash},
				),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: translateAccessModes(volumeRequest.AccessModes),
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{},
				},
			},
		}

		if appInstance.Generation > 0 {
			pvc.Annotations[labels.AcornAppGeneration] = strconv.FormatInt(appInstance.Generation, 10)
		}

		if bind {
			pv, err := getPVForVolumeBinding(req, appInstance, volumeBinding)
			if err != nil {
				return nil, err
			}

			if pv.Labels[labels.AcornPublicName] != "" {
				pvc.Labels[labels.AcornPublicName] = pv.Labels[labels.AcornPublicName]
			}

			pvc.Name = bindName(vol)
			pvc.Spec.VolumeName = pv.Name
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MinSize

			volumeClassName := volumeBinding.Class
			if volumeClassName == "" {
				volumeClassName = pv.Labels[labels.AcornVolumeClass]
			}

			volClass, ok := volumeClasses[volumeClassName]
			if !ok {
				return nil, fmt.Errorf("%s has an invalid volume class %s", vol, volumeBinding.Class)
			}
			pvc.Spec.StorageClassName = &volClass.StorageClassName
			pvc.Labels[labels.AcornVolumeClass] = volClass.Name

			if volumeBinding.Size != "" {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MustParseResourceQuantity(volumeBinding.Size)
			}
		} else {
			if volumeRequest.Class != "" {
				// Specifically allowing volume classes that are inactive.
				volClass, ok := volumeClasses[volumeRequest.Class]
				if !ok {
					return nil, fmt.Errorf("%s has an invalid volume class %s", vol, volumeRequest.Class)
				}
				pvc.Spec.StorageClassName = &volClass.StorageClassName
				pvc.Labels[labels.AcornVolumeClass] = volClass.Name
			}
			pvName, err := LookupExistingPV(req, appInstance, vol)
			if err != nil {
				return nil, err
			}
			pvc.Spec.VolumeName = pvName

			if volumeRequest.Size == "" {
				// If the volumeRequest does not have a size set, then we need to determine what default to use. If status.Defaults.Volumes has this
				// volume set, then we use that. Otherwise, we use v1 package level default size.
				defaultSize := *v1.DefaultSize
				if defaultVolume, hasDefaultSet := appInstance.Status.Defaults.Volumes[vol]; hasDefaultSet {
					defaultSize, err = resource.ParseQuantity(string(defaultVolume.Size))
					if err != nil {
						return nil, fmt.Errorf("failed to parse default volume size %q for volume %q: %w", defaultVolume.Size, vol, err)
					}
				}
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = defaultSize
			} else {
				pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *v1.MustParseResourceQuantity(volumeRequest.Size)
			}
		}

		// Ensure that no other PersistentVolume exists with the same public name
		selector := klabels.SelectorFromSet(map[string]string{
			labels.AcornPublicName: pvc.Labels[labels.AcornPublicName],
			// Make sure we only list volumes that are part of the same Acorn project
			labels.AcornAppNamespace: appInstance.Namespace,
		})
		pvList := new(corev1.PersistentVolumeList)
		if err := req.Client.List(req.Ctx, pvList, &kclient.ListOptions{LabelSelector: selector}); err != nil {
			return nil, err
		}

		if len(pvList.Items) > 1 || (len(pvList.Items) == 1 && pvList.Items[0].Name != pvc.Spec.VolumeName) {
			return nil, fmt.Errorf("a volume with the name %q already exists", pvc.Labels[labels.AcornPublicName])
		}

		result = append(result, &pvc)
	}
	return
}

func getPVForVolumeBinding(req router.Request, appInstance *v1.AppInstance, binding v1.VolumeBinding) (*corev1.PersistentVolume, error) {
	// binding.Volume can either be the actual name of the PersistentVolume, or its public name in Acorn.
	// Check for the actual name first.
	pv := new(corev1.PersistentVolume)
	if err := req.Client.Get(req.Ctx, kclient.ObjectKey{Name: binding.Volume}, pv); err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	} else if err == nil {
		// Make sure this PV is managed by acorn
		if _, ok := pv.Labels[labels.AcornManaged]; !ok {
			return nil, fmt.Errorf("no Acorn-managed volume found with name %q in project %q", binding.Volume, appInstance.Namespace)
		}

		// Make sure this PV has the Acorn app namespace label with the same value as the AppInstance's namespace
		if appNamespace, ok := pv.Labels[labels.AcornAppNamespace]; !ok || appNamespace != appInstance.Namespace {
			return nil, fmt.Errorf("no Acorn-managed volume found with name %q in project %q", binding.Volume, appInstance.Namespace)
		}

		// Check the PV phase
		if !isPVAvailable(pv, appInstance.Name) {
			return nil, fmt.Errorf("volume %q is not available for binding", binding.Volume)
		}

		return pv, nil
	}

	// If we didn't find it by name, then look for it by public name.
	selector := klabels.SelectorFromSet(map[string]string{
		labels.AcornPublicName: binding.Volume,
		// Make sure we only list volumes that are part of the same Acorn project
		labels.AcornAppNamespace: appInstance.Namespace,
	})

	pvList := new(corev1.PersistentVolumeList)
	if err := req.Client.List(req.Ctx, pvList, &kclient.ListOptions{LabelSelector: selector}); err != nil {
		return nil, err
	}
	if len(pvList.Items) != 1 {
		if len(pvList.Items) == 0 {
			return nil, fmt.Errorf("no Acorn-managed volume found with name %q in project %q", binding.Volume, appInstance.Namespace)
		}
		return nil, fmt.Errorf("expected 1 PV for volume %s, found %d", binding.Volume, len(pvList.Items))
	}

	// Check the PV phase
	if !isPVAvailable(&pvList.Items[0], appInstance.Name) {
		return nil, fmt.Errorf("volume %q is not available for binding", binding.Volume)
	}

	return &pvList.Items[0], nil
}

func volumeLabels(appInstance *v1.AppInstance, volume string, volumeRequest v1.VolumeRequest) map[string]string {
	labelMap := map[string]string{
		labels.AcornAppName:      appInstance.Name,
		labels.AcornAppNamespace: appInstance.Namespace,
		labels.AcornManaged:      "true",
		labels.AcornVolumeName:   volume,
		labels.AcornPublicName:   publicname.ForChild(appInstance, volume),
	}
	return labels.Merge(labelMap, labels.GatherScoped(volume, v1.LabelTypeVolume, appInstance.Status.AppSpec.Labels,
		volumeRequest.Labels, appInstance.Spec.Labels))
}

func isEphemeral(appInstance *v1.AppInstance, volume string, addHelperVolume bool) (v1.VolumeRequest, bool) {
	if volume == AcornHelper && (appInstance.Status.GetDevMode() || addHelperVolume) {
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
		if volume.Preload {
			volumeReferences[volumeReference{name: AcornHelper}] = true
		}
		if volume.ContextDir != "" {
			if app.Status.GetDevMode() {
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
		if file.Secret.Name != "" && !strings.Contains(file.Secret.Name, ".") {
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
		} else if file.Content == "" && strings.Contains(file.Secret.Name, ".") {
			fileModes[volume.NormalizeMode(file.Mode)] = true
		}
	}
	for _, sidecar := range container.Sidecars {
		for _, file := range sidecar.Files {
			if file.Content != "" && file.Secret.Name == "" {
				fileModes[volume.NormalizeMode(file.Mode)] = true
			} else if file.Content == "" && strings.Contains(file.Secret.Name, ".") {
				fileModes[volume.NormalizeMode(file.Mode)] = true
			}
		}
	}
}

func secretPodVolName(secretName string) string {
	return strings.ReplaceAll(name.SafeConcatName("secret-", secretName), ".", "-")
}

func toVolumes(appInstance *v1.AppInstance, container v1.Container, interpolator *secrets.Interpolator, addWaitVolume, addHelperVolume bool) (result []corev1.Volume, _ error) {
	hasPorts := len(container.Ports) > 0
	volumeReferences := map[volumeReference]bool{}
	addVolumeReferencesForContainer(appInstance, volumeReferences, container)
	for _, entry := range typed.Sorted(container.Sidecars) {
		addVolumeReferencesForContainer(appInstance, volumeReferences, entry.Value)
		hasPorts = hasPorts || len(entry.Value.Ports) > 0
	}

	for volume := range volumeReferences {
		if volume.secretName != "" {
			mode, err := volume.ParseMode()
			if err != nil {
				return nil, err
			}
			result = append(result, corev1.Volume{
				Name: secretPodVolName(volume.secretName + volume.Suffix()),
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
		if vr, ok := isEphemeral(appInstance, volume.name, addHelperVolume); ok && !bind {
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

	if addWaitVolume && hasPorts {
		result = append(result, corev1.Volume{
			Name: string(appInstance.UID),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: z.Pointer[int32](0744),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: string(appInstance.UID),
					},
				},
			},
		})
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

func isPVAvailable(pv *corev1.PersistentVolume, appName string) bool {
	return pv.Status.Phase == corev1.VolumeAvailable || pv.Status.Phase == corev1.VolumeReleased ||
		pv.Labels[labels.AcornAppName] == appName
}
