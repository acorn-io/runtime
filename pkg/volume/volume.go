package volume

import (
	"context"
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	"golang.org/x/exp/slices"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/kubectl/pkg/util/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SyncVolumeClasses(req router.Request, resp router.Response) error {
	storageClass := req.Object.(*storagev1.StorageClass)
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	// If the admin has chosen to manually manage the volume classes, then there is nothing to do.
	if *cfg.ManageVolumeClasses {
		return nil
	}

	resp.Objects(&adminv1.ClusterVolumeClassInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: storageClass.Name,
		},
		Description:      "Acorn-generated volume class representing the storage class " + storageClass.Name,
		StorageClassName: storageClass.Name,
		Default:          storageClass.Annotations[storage.IsDefaultStorageClassAnnotation] == "true",
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
		Description: "Acorn-generated volume class representing ephemeral volumes not backed by a storage class",
	})

	return nil
}

func GetVolumeClassNames(ctx context.Context, c client.Client, namespace string, storageClassNames bool) ([]string, error) {
	volumeClasses, _, err := GetVolumeClasses(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	return getVolumeClassNames(volumeClasses, storageClassNames), nil
}

// GetVolumeClasses returns an array of all project and cluster volume classes available in the namespace. If a project
// volume class is set to default, this ensures that no cluster volume classes are default to avoid conflicts.
// The class determined to be default, if it exists, is also returned.
func GetVolumeClasses(ctx context.Context, c client.Client, namespace string) (map[string]adminv1.ProjectVolumeClassInstance, *adminv1.ProjectVolumeClassInstance, error) {
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

func ValidateVolumeClasses(ctx context.Context, c client.Client, namespace string, appInstanceSpec v1.AppInstanceSpec, appSpec *v1.AppSpec) *field.Error {
	if len(appInstanceSpec.Volumes) == 0 && len(appSpec.Volumes) == 0 {
		return nil
	}

	volumeClasses, defaultVolumeClass, err := GetVolumeClasses(ctx, c, namespace)
	if err != nil {
		return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, err.Error())
	}

	volumeBindings := make(map[string]v1.VolumeBinding)
	for i, vol := range appInstanceSpec.Volumes {
		if volClass, ok := volumeClasses[vol.Class]; vol.Class != "" && (!ok || volClass.Inactive) {
			return field.Invalid(field.NewPath("spec", "volumes").Index(i), vol.Class, "not a valid volume class")
		}
		volumeBindings[vol.Target] = vol
	}

	var (
		volClass adminv1.ProjectVolumeClassInstance
		ok       bool
	)
	for name, vol := range appSpec.Volumes {
		calculatedVolumeRequest := CopyVolumeDefaults(vol, volumeBindings[name], v1.VolumeDefault{})
		if calculatedVolumeRequest.Class != "" {
			volClass, ok = volumeClasses[calculatedVolumeRequest.Class]
			if !ok || volClass.Inactive {
				return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, fmt.Sprintf("%s is not a valid volume class", calculatedVolumeRequest.Class))
			}
		} else if defaultVolumeClass != nil {
			volClass = *defaultVolumeClass
		} else {
			return field.Invalid(field.NewPath("spec", "image"), appInstanceSpec.Image, fmt.Sprintf("no volume class found for %s", name))
		}

		if calculatedVolumeRequest.Size != "" {
			q := v1.MustParseResourceQuantity(calculatedVolumeRequest.Size)
			if volClass.Size.Min != "" && q.Cmp(*v1.MustParseResourceQuantity(volClass.Size.Min)) < 0 {
				return field.Invalid(field.NewPath("spec", "volumes", name, "size"), q.String(), fmt.Sprintf("less than volume class %s minimum of %v", calculatedVolumeRequest.Class, volClass.Size.Min))
			}
			if volClass.Size.Max != "" && q.Cmp(*v1.MustParseResourceQuantity(volClass.Size.Max)) > 0 {
				return field.Invalid(field.NewPath("spec", "volumes", name, "size"), q.String(), fmt.Sprintf("greater than volume class %s maximum of %v", calculatedVolumeRequest.Class, volClass.Size.Max))
			}
		}
		if volClass.AllowedAccessModes != nil {
			for _, am := range calculatedVolumeRequest.AccessModes {
				if !slices.Contains(volClass.AllowedAccessModes, am) {
					return field.Invalid(field.NewPath("spec", "volumes", name, "accessModes"), am, fmt.Sprintf("not an allowed access mode of %v", calculatedVolumeRequest.Class))
				}
			}
		}
	}

	return nil
}

func CopyVolumeDefaults(volumeRequest v1.VolumeRequest, volumeBinding v1.VolumeBinding, volumeDefaults v1.VolumeDefault) v1.VolumeRequest {
	bind := volumeBinding.Volume != ""
	if volumeBinding.Class != "" {
		volumeRequest.Class = volumeBinding.Class
	} else if !bind && volumeRequest.Class == "" {
		volumeRequest.Class = volumeDefaults.Class
	}

	if volumeBinding.Size != "" {
		volumeRequest.Size = volumeBinding.Size
	} else if !bind && volumeRequest.Size == "" {
		volumeRequest.Size = volumeDefaults.Size
	}

	if len(volumeBinding.AccessModes) != 0 {
		volumeRequest.AccessModes = volumeBinding.AccessModes
	} else if len(volumeRequest.AccessModes) == 0 {
		volumeRequest.AccessModes = volumeDefaults.AccessModes
	}

	return volumeRequest
}
