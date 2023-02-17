package imageallowrules

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	internalv1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"

	"github.com/acorn-io/mink/pkg/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	return &apiv1.ImageAllowRulesList{}
}

func (s *Strategy) New() types.Object {
	return &apiv1.ImageAllowRules{}
}

func (s *Strategy) List(ctx context.Context, namespace string, options storage.ListOptions) (types.ObjectList, error) {

	imageAllowRulesList := &internalv1.ImageAllowRulesInstanceList{}
	if err := s.client.List(ctx, imageAllowRulesList, &client.ListOptions{Namespace: namespace}); err != nil {
		return nil, err
	}

	return imageAllowRulesList, nil
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

	imageAllowRules := list.(*apiv1.ImageAllowRulesList)
	for _, imageAllowRulesItem := range imageAllowRules.Items {
		if imageAllowRulesItem.Name == name {
			return &imageAllowRulesItem, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "imageallowrules",
	}, name)
}
