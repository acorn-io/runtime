package client

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const IsDefaultStorageClassAnnotation = "storageclass.kubernetes.io/is-default-class"

func pvToVolume(pv corev1.PersistentVolume) Volume {
	var (
		accessModes      []v1.AccessMode
		shortAccessModes []string
	)
	for _, accessMode := range pv.Spec.AccessModes {
		switch accessMode {
		case corev1.ReadWriteOnce:
			accessModes = append(accessModes, v1.AccessModeReadWriteOnce)
			shortAccessModes = append(shortAccessModes, "RWO")
		case corev1.ReadOnlyMany:
			accessModes = append(accessModes, v1.AccessModeReadOnlyMany)
			shortAccessModes = append(shortAccessModes, "ROX")
		case corev1.ReadWriteMany:
			accessModes = append(accessModes, v1.AccessModeReadWriteMany)
			shortAccessModes = append(shortAccessModes, "RWX")
		}
	}
	vol := Volume{
		Name:        pv.Name,
		Created:     pv.CreationTimestamp,
		Revision:    pv.ResourceVersion,
		Labels:      pv.Labels,
		Annotations: pv.Annotations,
		Capacity:    pv.Spec.Capacity.Storage(),
		AccessModes: accessModes,
		Class:       pv.Spec.StorageClassName,
		Status: VolumeStatus{
			AppName:      pv.Labels[labels.AcornAppName],
			AppNamespace: pv.Labels[labels.AcornAppNamespace],
			VolumeName:   pv.Labels[labels.AcornVolumeName],
			Status:       strings.ToLower(string(pv.Status.Phase)),
			Message:      pv.Status.Message,
			Reason:       pv.Status.Reason,
			Columns: VolumeColumns{
				AccessModes: strings.Join(shortAccessModes, ","),
			},
		},
	}
	if !pv.DeletionTimestamp.IsZero() {
		vol.Status.Status += "/deleted"
	}
	return vol
}

func (c *client) VolumeList(ctx context.Context) (result []Volume, _ error) {
	var sel klabels.Selector
	if c.Namespace == "" {
		rel, err := klabels.NewRequirement(labels.AcornAppNamespace, selection.Exists, nil)
		if err != nil {
			return nil, err
		}
		sel = klabels.SelectorFromSet(map[string]string{
			labels.AcornManaged: "true",
		}).Add(*rel)
	} else {
		sel = klabels.SelectorFromSet(map[string]string{
			labels.AcornAppNamespace: c.Namespace,
			labels.AcornManaged:      "true",
		})
	}
	pvs := &corev1.PersistentVolumeList{}
	err := c.Client.List(ctx, pvs, &kclient.ListOptions{
		LabelSelector: sel,
	})
	if err != nil {
		return nil, err
	}

	for _, pv := range pvs.Items {
		result = append(result, pvToVolume(pv))
	}

	return result, nil
}

func (c *client) VolumeGet(ctx context.Context, name string) (*Volume, error) {
	pv := &corev1.PersistentVolume{}
	err := c.Client.Get(ctx, kclient.ObjectKey{
		Name: name,
	}, pv)
	if err != nil {
		return nil, err
	}

	if c.Namespace != "" && pv.Labels[labels.AcornAppNamespace] != c.Namespace {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    "acorn.io",
			Resource: "volumes",
		}, name)
	}

	vol := pvToVolume(*pv)
	return &vol, nil
}

func (c *client) VolumeDelete(ctx context.Context, name string) (*Volume, error) {
	// get first to ensure the namespace matches
	v, err := c.VolumeGet(ctx, name)
	if apierror.IsNotFound(err) {
		return nil, nil
	}
	return v, c.Client.Delete(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})
}

func (c *client) VolumeCreate(ctx context.Context, name string, capacity resource.Quantity, opts *VolumeCreateOptions) (*Volume, error) {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: capacity,
			},
			PersistentVolumeSource: corev1.PersistentVolumeSource{},
		},
	}

	if strings.HasSuffix(name, "-") {
		pv.Name = ""
		pv.GenerateName = pv.Name
	}

	if opts != nil {
		for _, accessMode := range opts.AccessModes {
			switch accessMode {
			case v1.AccessModeReadWriteOnce:
				pv.Spec.AccessModes = append(pv.Spec.AccessModes, corev1.ReadWriteOnce)
			case v1.AccessModeReadOnlyMany:
				pv.Spec.AccessModes = append(pv.Spec.AccessModes, corev1.ReadOnlyMany)
			case v1.AccessModeReadWriteMany:
				pv.Spec.AccessModes = append(pv.Spec.AccessModes, corev1.ReadWriteMany)
			default:
				return nil, fmt.Errorf("unknown access mode: %s", accessMode)
			}
		}
	}

	if opts != nil && opts.Class != "" {
		pv.Spec.StorageClassName = opts.Class
	} else {
		storageClasses := &storagev1.StorageClassList{}
		err := c.Client.List(ctx, storageClasses)
		if err != nil {
			return nil, err
		}
		for _, class := range storageClasses.Items {
			if class.Annotations[IsDefaultStorageClassAnnotation] == "true" {
				pv.Spec.StorageClassName = class.Name
			}
		}

		if pv.Spec.StorageClassName == "" {
			return nil, fmt.Errorf("no class set and failed to find a default class")
		}
	}

	err := c.Client.Create(ctx, pv)
	if err != nil {
		return nil, err
	}

	vol := pvToVolume(*pv)
	return &vol, nil
}

func ToAccessModes(accessModes []string) (result []v1.AccessMode, _ error) {
	for _, accessMode := range accessModes {
		switch strings.ToLower(accessMode) {
		case "readwriteonce", "rwo":
			result = append(result, v1.AccessModeReadWriteOnce)
		case "readonlymany", "rom", "rox":
			result = append(result, v1.AccessModeReadOnlyMany)
		case "readwritemany", "rwm", "rwx":
			result = append(result, v1.AccessModeReadWriteMany)
		default:
			return nil, fmt.Errorf("unknown access mode: %s", accessMode)
		}
	}
	return
}
