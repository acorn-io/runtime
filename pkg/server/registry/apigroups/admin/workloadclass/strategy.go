package workloadclass

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	adminv1 "github.com/acorn-io/acorn/pkg/apis/internal.admin.acorn.io/v1"

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
	return &apiv1.WorkloadClassList{}
}

func (s *Strategy) New() types.Object {
	return &apiv1.WorkloadClass{}
}

func (s *Strategy) List(ctx context.Context, namespace string, options storage.ListOptions) (types.ObjectList, error) {
	clusterWorkloadClasses := &adminv1.ClusterWorkloadClassInstanceList{}
	if err := s.client.List(ctx, clusterWorkloadClasses); err != nil {
		return nil, err
	}

	projectWorkloadClasses := &adminv1.ProjectWorkloadClassInstanceList{}
	if err := s.client.List(ctx, projectWorkloadClasses, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	workloadClasses := apiv1.WorkloadClassList{Items: make(
		[]apiv1.WorkloadClass,
		0,
		len(clusterWorkloadClasses.Items)+len(projectWorkloadClasses.Items))}

	var projectDefaultExists bool
	projectWorkloadClassesSeen := make(map[string]struct{})
	for _, pwc := range projectWorkloadClasses.Items {
		if pwc.Default {
			projectDefaultExists = true
		}
		workloadClasses.Items = append(workloadClasses.Items, apiv1.WorkloadClass{
			ObjectMeta:  v1.ObjectMeta{Name: pwc.Name, Namespace: pwc.Namespace, CreationTimestamp: pwc.CreationTimestamp},
			Memory:      adminv1.WorkloadClassMemory(pwc.Memory),
			Default:     pwc.Default,
			Description: pwc.Description,
		})
		projectWorkloadClassesSeen[pwc.Name] = struct{}{}
	}

	for _, cwc := range clusterWorkloadClasses.Items {
		if _, ok := projectWorkloadClassesSeen[cwc.Name]; ok {
			// A project volume class with the same name exists, skipping the cluster volume class
			continue
		}
		if projectDefaultExists {
			cwc.Default = false
		}
		workloadClasses.Items = append(workloadClasses.Items, apiv1.WorkloadClass{
			ObjectMeta:  v1.ObjectMeta{Name: cwc.Name},
			Memory:      adminv1.WorkloadClassMemory(cwc.Memory),
			Default:     cwc.Default,
			Description: cwc.Description,
		})
	}

	return &workloadClasses, nil
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

	workloadClasses := list.(*apiv1.WorkloadClassList)
	for _, workloadClass := range workloadClasses.Items {
		if workloadClass.Name == name {
			return &workloadClass, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "workloadclass",
	}, name)
}
