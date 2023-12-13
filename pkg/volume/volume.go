package volume

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/util/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NormalizeMode(mode string) string {
	if mode == "0644" || mode == "644" {
		return ""
	}
	return mode
}

func SyncVolumeClasses(req router.Request, resp router.Response) error {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	// If the admin has chosen to manually manage the volume classes or the storage class has been deleted, then there is nothing to do.
	if *cfg.ManageVolumeClasses || req.Object == nil || !req.Object.GetDeletionTimestamp().IsZero() {
		return nil
	}

	storageClass := req.Object.(*storagev1.StorageClass)
	resp.Objects(&adminv1.ClusterVolumeClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClass.Name,
		},
		Description:      "Acorn-generated volume class representing the storage class " + storageClass.Name,
		StorageClassName: storageClass.Name,
		Default:          storageClass.Annotations[storage.IsDefaultStorageClassAnnotation] == "true",
		SupportedRegions: []string{apiv1.LocalRegion},
	})

	return nil
}

func CreateEphemeralVolumeClass(req router.Request, resp router.Response) error {
	cfg, err := config.UnmarshalAndComplete(req.Ctx, req.Object.(*corev1.ConfigMap), req.Client)
	if err != nil || *cfg.ManageVolumeClasses {
		return err
	}

	resp.Objects(&adminv1.ClusterVolumeClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ephemeral",
		},
		Description:      "Acorn-generated volume class representing ephemeral volumes not backed by a storage class",
		SupportedRegions: []string{apiv1.LocalRegion},
	})

	return nil
}

func GetVolumeClassNames(ctx context.Context, c client.Client, namespace string, storageClassNames bool) ([]string, error) {
	volumeClasses, _, err := GetVolumeClassInstances(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	return getVolumeClassNames(volumeClasses, storageClassNames), nil
}

// GetVolumeClassInstances returns an array of all project and cluster volume classes available in the namespace. If a project
// volume class is set to default, this ensures that no cluster volume classes are default to avoid conflicts.
// The class determined to be default, if it exists, is also returned.
func GetVolumeClassInstances(ctx context.Context, c client.Client, namespace string) (map[string]adminv1.ProjectVolumeClassInstance, *adminv1.ProjectVolumeClassInstance, error) {
	volumeClasses := new(adminv1.ProjectVolumeClassInstanceList)
	if err := c.List(ctx, volumeClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, nil, err
	}

	var defaultVolumeClass *adminv1.ProjectVolumeClassInstance
	var projectDefaultFound bool
	projectClassesSeen := make(map[string]struct{}, len(volumeClasses.Items))
	for i, vc := range volumeClasses.Items {
		if vc.Default {
			if !vc.Inactive {
				projectDefaultFound = true
				// Ordering of the default volume class name ensure our error messages don't flop.
				if defaultVolumeClass == nil || vc.Name < defaultVolumeClass.Name {
					defaultVolumeClass = vc.DeepCopy()
				}
			} else {
				vc.Default = false
				volumeClasses.Items[i] = vc
			}
		}
		projectClassesSeen[vc.Name] = struct{}{}
	}

	clusterVolumeClasses := new(adminv1.ClusterVolumeClassInstanceList)
	if err := c.List(ctx, clusterVolumeClasses); err != nil {
		return nil, nil, err
	}

	for _, cvc := range clusterVolumeClasses.Items {
		if _, ok := projectClassesSeen[cvc.Name]; ok {
			// Project volume class with the same name exists, skipping cluster volume class
			continue
		}
		if cvc.Default {
			if projectDefaultFound || cvc.Inactive {
				cvc.Default = false
			} else if defaultVolumeClass == nil || cvc.Name < defaultVolumeClass.Name {
				// Ordering of the default volume class name ensure our error messages don't flop.
				defaultVolumeClass = (*adminv1.ProjectVolumeClassInstance)(cvc.DeepCopy())
			}
		}
		volumeClasses.Items = append(volumeClasses.Items, adminv1.ProjectVolumeClassInstance(cvc))
	}

	return SliceToMap(volumeClasses.Items, func(obj adminv1.ProjectVolumeClassInstance) string {
			return obj.Name
		}),
		defaultVolumeClass,
		nil
}

func SliceToMap[T any, K comparable](s []T, keyFunc func(obj T) K) map[K]T {
	m := make(map[K]T, len(s))
	for _, obj := range s {
		m[keyFunc(obj)] = obj
	}

	return m
}

func getVolumeClassNames(volumeClasses map[string]adminv1.ProjectVolumeClassInstance, storageClassNames bool) []string {
	if !storageClassNames {
		return typed.SortedKeys(volumeClasses)
	}
	storageClassName := make(map[string]struct{}, len(volumeClasses))
	for _, sc := range volumeClasses {
		storageClassName[sc.StorageClassName] = struct{}{}
	}

	return typed.SortedKeys(storageClassName)
}

func ResolveVolumeRequest(ctx context.Context, c client.Client, volumeRequest v1.VolumeRequest,
	volumeBinding v1.VolumeBinding, volumeClasses map[string]adminv1.ProjectVolumeClassInstance,
	defaultVolumeClass *adminv1.ProjectVolumeClassInstance, existingResolvedVolume v1.VolumeResolvedOffering) (v1.VolumeRequest, error) {
	bind := volumeBinding.Volume != ""
	trueVolumeClass := defaultVolumeClass.DeepCopy()

	if volumeBinding.Class != "" {
		volumeRequest.Class = volumeBinding.Class
		vc := volumeClasses[volumeBinding.Class]
		trueVolumeClass = &vc
	} else if volumeRequest.Class != "" {
		vc := volumeClasses[volumeRequest.Class]
		trueVolumeClass = &vc
	} else if !bind && defaultVolumeClass != nil {
		volumeRequest.Class = defaultVolumeClass.Name
	}

	if volumeBinding.Size != "" {
		volumeRequest.Size = volumeBinding.Size
	} else if !bind && volumeRequest.Size == "" {
		if existingResolvedVolume.Size != "" {
			volumeRequest.Size = existingResolvedVolume.Size
		} else if trueVolumeClass != nil {
			volumeRequest.Size = trueVolumeClass.Size.Default
		} else {
			defaultSize, err := GetDefaultVolumeSize(ctx, c)
			if err != nil {
				return v1.VolumeRequest{}, err
			}
			volumeRequest.Size = defaultSize
		}
	}

	if len(volumeBinding.AccessModes) > 0 {
		volumeRequest.AccessModes = volumeBinding.AccessModes
	} else if !bind && len(volumeRequest.AccessModes) == 0 && trueVolumeClass != nil {
		volumeRequest.AccessModes = trueVolumeClass.AllowedAccessModes
	}

	// If there is an existing VolumeResolvedOffering, and we are not binding to an already existing volume,
	// then make sure that we continue using the same VolumeClass and AccessModes, since those cannot be
	// changed on existing volumes
	if !bind {
		if existingResolvedVolume.Class != "" {
			volumeRequest.Class = existingResolvedVolume.Class
		}
		if len(existingResolvedVolume.AccessModes) > 0 {
			volumeRequest.AccessModes = existingResolvedVolume.AccessModes
		}
	}

	return volumeRequest, nil
}

func FindDefaultStorageClass(ctx context.Context, c client.Reader) (string, error) {
	storageClasses := &storagev1.StorageClassList{}
	if err := c.List(ctx, storageClasses); err != nil {
		return "", err
	}

	for _, sc := range storageClasses.Items {
		if sc.Annotations[storage.IsDefaultStorageClassAnnotation] == "true" {
			return sc.Name, nil
		}
	}

	return "", nil
}

func GetDefaultVolumeSize(ctx context.Context, c client.Client) (v1.Quantity, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return "", err
	}

	// If the default volume size is set in the config, use that. Otherwise use the
	// package level default in internalv1.
	defaultVolumeSize := v1.DefaultSizeQuantity
	if cfg.VolumeSizeDefault != "" {
		defaultVolumeSize = v1.Quantity(cfg.VolumeSizeDefault)
	}

	return defaultVolumeSize, nil
}
