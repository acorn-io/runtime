package computeclass

import (
	"context"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	adminv1 "github.com/acorn-io/runtime/pkg/apis/internal.admin.acorn.io/v1"

	"github.com/acorn-io/mink/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client client.WithWatch
}

func NewStrategy(c client.WithWatch) *Strategy {
	return &Strategy{client: c}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.ComputeClassList{}
}

func (s *Strategy) New() types.Object {
	return &apiv1.ComputeClass{}
}

func (s *Strategy) List(ctx context.Context, namespace string, options storage.ListOptions) (types.ObjectList, error) {
	clusterComputeClasses := &adminv1.ClusterComputeClassInstanceList{}
	if err := s.client.List(ctx, clusterComputeClasses); err != nil {
		return nil, err
	}

	projectComputeClasses := &adminv1.ProjectComputeClassInstanceList{}
	if err := s.client.List(ctx, projectComputeClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	computeClasses := apiv1.ComputeClassList{Items: make(
		[]apiv1.ComputeClass,
		0,
		len(clusterComputeClasses.Items)+len(projectComputeClasses.Items))}

	var projectDefaultExists bool
	projectComputeClassesSeen := make(map[string]struct{})
	for _, pcc := range projectComputeClasses.Items {
		if pcc.Default {
			projectDefaultExists = true
		}
		computeClasses.Items = append(computeClasses.Items, apiv1.ComputeClass{
			ObjectMeta:       v1.ObjectMeta{Name: pcc.Name, Namespace: pcc.Namespace, CreationTimestamp: pcc.CreationTimestamp},
			Memory:           pcc.Memory,
			Default:          pcc.Default,
			Description:      pcc.Description,
			SupportedRegions: pcc.SupportedRegions,
		})
		projectComputeClassesSeen[pcc.Name] = struct{}{}
	}

	for _, ccc := range clusterComputeClasses.Items {
		if _, ok := projectComputeClassesSeen[ccc.Name]; ok {
			// A project volume class with the same name exists, skipping the cluster volume class
			continue
		}
		if projectDefaultExists {
			ccc.Default = false
		}
		computeClasses.Items = append(computeClasses.Items, apiv1.ComputeClass{
			ObjectMeta:       v1.ObjectMeta{Name: ccc.Name},
			Memory:           ccc.Memory,
			Default:          ccc.Default,
			Description:      ccc.Description,
			SupportedRegions: ccc.SupportedRegions,
		})
	}

	return &computeClasses, nil
}

func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	list, err := s.List(ctx, namespace, storage.ListOptions{
		Predicate: storage.SelectionPredicate{
			Field: fields.SelectorFromSet(fields.Set{"metadata.name": name}),
		},
	})
	if err != nil {
		return nil, err
	}

	computeClasses := list.(*apiv1.ComputeClassList)
	for _, computeClass := range computeClasses.Items {
		if computeClass.Name == name {
			return &computeClass, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "computeclass",
	}, name)
}
