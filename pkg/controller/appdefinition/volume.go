package appdefinition

import (
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/meta"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	name2 "github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func toPVCs(appInstance *v1.AppInstance) (result []meta.Object) {
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

func addVolumeReferencesForContainer(volumeNames map[string]bool, container v1.Container) {
	for _, volume := range container.Dirs {
		if volume.ContextDir != "" {
			continue
		}
		if volume.Secret.Name == "" {
			volumeNames[volume.Volume] = true
		} else {
			volumeNames["secret--"+volume.Secret.Name] = true
		}
	}

	for _, file := range container.Files {
		if file.Secret.Name != "" {
			volumeNames["secret--"+file.Secret.Name] = true
		}
	}
}

func isSecretOptional(appInstance *v1.AppInstance, secretName string) *bool {
	opt := appInstance.Status.AppSpec.Secrets[secretName].Optional
	b := opt != nil && *opt
	return &b
}

func toVolumes(appInstance *v1.AppInstance, container v1.Container) (result []corev1.Volume) {
	volumeNames := map[string]bool{}
	addVolumeReferencesForContainer(volumeNames, container)
	for _, sidecar := range container.Sidecars {
		addVolumeReferencesForContainer(volumeNames, sidecar)
	}

	for _, volume := range typed.SortedKeys(volumeNames) {
		if strings.HasPrefix(volume, "secret--") {
			secretName := strings.TrimPrefix(volume, "secret--")
			result = append(result, corev1.Volume{
				Name: volume,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: secretName,
						Optional:   isSecretOptional(appInstance, secretName),
					},
				},
			})
			continue
		}

		name, bind := toVolumeName(appInstance, volume)
		if vr, ok := isEphemeral(appInstance, volume); ok && !bind {
			result = append(result, corev1.Volume{
				Name: volume,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: resource.NewQuantity(vr.Size*1_000_000_000, resource.DecimalSI),
					},
				},
			})
		} else {
			result = append(result, corev1.Volume{
				Name: volume,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: name,
					},
				},
			})
		}
	}

	for _, file := range container.Files {
		if file.Content != "" && file.Secret.Name == "" {
			result = append(result, corev1.Volume{
				Name: "files",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "files",
						},
					},
				},
			})
			break
		}
	}

	return
}
