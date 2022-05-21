package volumes

import (
	"context"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	api "github.com/acorn-io/acorn/pkg/apis/api.acorn.io"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	kclient "github.com/acorn-io/acorn/pkg/k8sclient"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/namespace"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/acorn/pkg/watcher"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewStorage(c client.WithWatch) *Storage {
	return &Storage{
		TableConvertor: tables.VolumeConverter,
		client:         c,
	}
}

type Storage struct {
	rest.TableConvertor

	client client.WithWatch
}

func (s *Storage) NewList() runtime.Object {
	return &apiv1.VolumeList{}
}

func (s *Storage) NamespaceScoped() bool {
	return true
}

func (s *Storage) New() runtime.Object {
	return &apiv1.Volume{}
}

func (s *Storage) namespaceRequirement(ctx context.Context, nsName string) (*klabels.Requirement, error) {
	if nsName == "" {
		return klabels.NewRequirement(labels.AcornAppNamespace, selection.Exists, nil)
	}

	nses, err := namespace.Descendants(ctx, s.client, nsName)
	if err != nil {
		return nil, err
	}

	return klabels.NewRequirement(labels.AcornAppNamespace, selection.In, nses.List())
}

func (s *Storage) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	var (
		sel       klabels.Selector
		nsName, _ = request.NamespaceFrom(ctx)
	)

	rel, err := s.namespaceRequirement(ctx, nsName)
	if err != nil {
		return nil, err
	}
	sel = klabels.SelectorFromSet(map[string]string{
		labels.AcornManaged: "true",
	}).Add(*rel)

	pvs := &corev1.PersistentVolumeList{}
	err = s.client.List(ctx, pvs, &kclient.ListOptions{
		LabelSelector: sel,
	})
	if err != nil {
		return nil, err
	}

	result := &apiv1.VolumeList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: pvs.ResourceVersion,
		},
	}

	for _, pv := range pvs.Items {
		result.Items = append(result.Items, *pvToVolume(pv, nsName))
	}

	return result, nil
}

func (s *Storage) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	pv := &corev1.PersistentVolume{}
	err := s.client.Get(ctx, kclient.ObjectKey{
		Name: name,
	}, pv)
	if err != nil {
		return nil, err
	}

	if pv.Labels[labels.AcornAppNamespace] == "" {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "volumes",
		}, name)
	}

	parent, err := namespace.ParentMost(ctx, s.client, pv.Labels[labels.AcornAppNamespace])
	if err != nil {
		return nil, err
	}

	if ns != "" && ns != parent.Name {
		return nil, apierror.NewNotFound(schema.GroupResource{
			Group:    api.Group,
			Resource: "volumes",
		}, name)
	}

	return pvToVolume(*pv, parent.Name), nil
}

func (s *Storage) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	// get first to ensure the namespace matches
	v, err := s.Get(ctx, name, nil)
	if err != nil {
		return nil, false, err
	}
	if deleteValidation != nil {
		if err := deleteValidation(ctx, v); err != nil {
			return nil, false, err
		}
	}

	return v, true, s.client.Delete(ctx, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	})
}

func (s *Storage) Watch(ctx context.Context, options *internalversion.ListOptions) (watch.Interface, error) {
	ns, _ := request.NamespaceFrom(ctx)
	w, err := s.client.Watch(ctx, &corev1.PersistentVolumeList{}, watcher.ListOptions("", options))
	if err != nil {
		return nil, err
	}

	return watcher.Transform(w, func(obj runtime.Object) []runtime.Object {
		pv := obj.(*corev1.PersistentVolume)
		appNamespace := pv.Labels[labels.AcornAppNamespace]
		if appNamespace == "" {
			return nil
		}
		parent, err := namespace.ParentMost(ctx, s.client, appNamespace)
		if err != nil {
			return nil
		}
		vol := pvToVolume(*pv, parent.Name)
		if ns == "" || ns == vol.Name {
			return []runtime.Object{vol}
		}
		return nil
	}), nil
}

func pvToVolume(pv corev1.PersistentVolume, namespace string) *apiv1.Volume {
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

	vol := &apiv1.Volume{
		ObjectMeta: pv.ObjectMeta,
		Spec: apiv1.VolumeSpec{
			Capacity:    pv.Spec.Capacity.Storage(),
			AccessModes: accessModes,
			Class:       pv.Spec.StorageClassName,
		},
		Status: apiv1.VolumeStatus{
			AppName:      pv.Labels[labels.AcornAppName],
			AppNamespace: pv.Labels[labels.AcornAppNamespace],
			VolumeName:   pv.Labels[labels.AcornVolumeName],
			Status:       strings.ToLower(string(pv.Status.Phase)),
			Columns: apiv1.VolumeColumns{
				AccessModes: strings.Join(shortAccessModes, ","),
			},
		},
	}
	vol.Namespace = namespace
	if !pv.DeletionTimestamp.IsZero() {
		vol.Status.Status += "/deleted"
	}

	return vol
}
