package appdefinition

import (
	"sort"
	"strconv"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	name2 "github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AcornHelper     = " /acorn-helper"
	AcornHelperPath = "/.acorn"
)

func addPVCs(appInstance *v1.AppInstance, resp router.Response) {
	resp.Objects(toPVCs(appInstance)...)
}

func translateAccessModes(accessModes []v1.AccessMode) (result []corev1.PersistentVolumeAccessMode) {
	for _, accessMode := range accessModes {
		newMode := strings.ToUpper(string(accessMode[0:1])) + string(accessMode[1:])
		result = append(result, corev1.PersistentVolumeAccessMode(newMode))
	}
	return
}

func toPVCs(appInstance *v1.AppInstance) (result []kclient.Object) {
	for _, entry := range typed.Sorted(appInstance.Status.AppSpec.Volumes) {
		volume, volumeRequest := entry.Key, entry.Value

		var (
			accessModes         = translateAccessModes(volumeRequest.AccessModes)
			volumeBinding, bind = isBind(appInstance, volume)
			class               *string
		)

		if volumeRequest.Class == v1.VolumeRequestTypeEphemeral && !bind {
			continue
		}

		pvc := corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      volume,
				Namespace: appInstance.Status.Namespace,
				Labels: map[string]string{
					labels.AcornAppName:      appInstance.Name,
					labels.AcornAppNamespace: appInstance.Namespace,
					labels.AcornManaged:      "true",
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: *resource.NewQuantity(volumeRequest.Size*1_000_000_000, resource.DecimalSI),
					},
				},
			},
		}

		if bind {
			pvc.Name = bindName(volume)
			pvc.Spec.VolumeName = volumeBinding.Volume
		} else {
			if volumeRequest.Class != "" {
				class = &volumeRequest.Class
			}
			pvc.Spec.StorageClassName = class
			if volumeBinding.Class != "" {
				pvc.Spec.StorageClassName = &volumeBinding.Class
			}

			if len(volumeBinding.AccessModes) > 0 {
				pvc.Spec.AccessModes = translateAccessModes(volumeBinding.AccessModes)
			}
		}

		if !volumeBinding.Capacity.IsZero() {
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = volumeBinding.Capacity
		}

		result = append(result, &pvc)
	}
	return
}

func isEphemeral(appInstance *v1.AppInstance, volume string) (v1.VolumeRequest, bool) {
	if volume == AcornHelper && appInstance.Spec.GetDevMode() {
		return v1.VolumeRequest{
			Class: v1.VolumeRequestTypeEphemeral,
			Size:  10,
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
		if v.VolumeRequest == volume {
			return v, true
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
	if normalizeMode(v.mode) == "" {
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
	if normalizeMode(v.mode) == "" {
		return nil, nil
	}
	return toMode(v.mode)
}

func getFilesFileModesForApp(app *v1.AppInstance) map[string]bool {
	fileModes := map[string]bool{}
	for _, container := range app.Status.AppSpec.Containers {
		addFilesFileModesForContainer(fileModes, container)
	}
	for _, container := range app.Status.AppSpec.Jobs {
		addFilesFileModesForContainer(fileModes, container)
	}
	return fileModes
}

func normalizeMode(mode string) string {
	if mode == "0644" || mode == "644" {
		return ""
	}
	return mode
}

func addFilesFileModesForContainer(fileModes map[string]bool, container v1.Container) {
	for _, file := range container.Files {
		if file.Content != "" && file.Secret.Name == "" {
			fileModes[normalizeMode(file.Mode)] = true
		}
		for _, sidecar := range container.Sidecars {
			for _, file := range sidecar.Files {
				if file.Content != "" && file.Secret.Name == "" {
					fileModes[normalizeMode(file.Mode)] = true
				}
			}
		}
	}
}

func toVolumes(appInstance *v1.AppInstance, container v1.Container) (result []corev1.Volume, _ error) {
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
						SizeLimit: resource.NewQuantity(vr.Size*1_000_000_000, resource.DecimalSI),
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

	for _, modeString := range typed.SortedKeys(fileModes) {
		name := "files"
		var (
			mode *int32
			err  error
		)
		if modeString != "" {
			name = "files-" + modeString
			mode, err = toMode(modeString)
			if err != nil {
				return nil, err
			}
		}
		result = append(result, corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: mode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
				},
			},
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return
}
