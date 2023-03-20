package class

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	admininternalv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client client.WithWatch
}

// Get is based on list because list will do the RBAC checks to ensure the user can access that
// project. This also ensure that you can only delete a project that you have namespace access to
func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	volumeClasses, err := s.list(ctx, namespace, storage.ListOptions{
		Predicate: storage.SelectionPredicate{
			Field: fields.SelectorFromSet(fields.Set{"metadata.name": name}),
		},
	})
	if err != nil {
		return nil, err
	}
	for _, volumeClass := range volumeClasses.Items {
		if volumeClass.Name == name {
			return &volumeClass, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "volumeclasses",
	}, name)
}

func (s *Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	return s.list(ctx, namespace, opts)
}

func (s *Strategy) list(ctx context.Context, namespace string, opts storage.ListOptions) (*apiv1.VolumeClassList, error) {
	var projectDefaultExists bool
	volumeClasses := new(apiv1.VolumeClassList)

	projectVolumeClasses := new(admininternalv1.ProjectVolumeClassInstanceList)
	if err := s.client.List(ctx, projectVolumeClasses, strategy.ToListOpts(namespace, opts)); err != nil {
		return nil, err
	}

	clusterVolumeClasses := new(admininternalv1.ClusterVolumeClassInstanceList)
	if err := s.client.List(ctx, clusterVolumeClasses, strategy.ToListOpts("", opts)); err != nil {
		return nil, err
	}

	volumeClasses.Items = make([]apiv1.VolumeClass, 0, len(projectVolumeClasses.Items)+len(clusterVolumeClasses.Items))
	projectVolumeClassesSeen := make(map[string]struct{})
	for _, vc := range projectVolumeClasses.Items {
		if vc.Default {
			projectDefaultExists = true
		}
		// Reset TypeMeta so proper GVK is reported
		vc.TypeMeta = metav1.TypeMeta{}
		volumeClasses.Items = append(volumeClasses.Items, apiv1.VolumeClass(vc))
		projectVolumeClassesSeen[vc.Name] = struct{}{}
	}

	for _, cvc := range clusterVolumeClasses.Items {
		if _, ok := projectVolumeClassesSeen[cvc.Name]; ok {
			// A project volume class with the same name exists, skipping the cluster volume class
			continue
		}
		if projectDefaultExists {
			cvc.Default = false
		}
		// Reset TypeMeta so proper GVK is reported
		cvc.TypeMeta = metav1.TypeMeta{}
		volumeClasses.Items = append(volumeClasses.Items, apiv1.VolumeClass(cvc))
	}

	return volumeClasses, nil
}

func (s *Strategy) New() types.Object {
	return new(apiv1.VolumeClass)
}

func (s *Strategy) NewList() types.ObjectList {
	return new(apiv1.VolumeClassList)
}
