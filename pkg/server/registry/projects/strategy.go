package secrets

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	c    kclient.Client
	next strategy.Lister
}

func (s *Strategy) allowed(ctx context.Context) (sets.String, bool, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, false, nil
	}

	crbs := &rbacv1.ClusterRoleBindingList{}
	err := s.c.List(ctx, crbs)
	if err != nil {
		return nil, false, err
	}

	result := sets.NewString()

	rulesName := sets.NewString()
	for _, crb := range crbs.Items {
		for _, subject := range crb.Subjects {
			switch subject.Kind {
			case "User":
				if subject.Name == user.GetName() {
					rulesName.Insert(crb.RoleRef.Name)
				}
			case "Group":
				if slices.Contains(user.GetGroups(), subject.Name) {
					rulesName.Insert(crb.RoleRef.Name)
				}
			}
		}
	}

	for _, ruleName := range rulesName.List() {
		rule := &rbacv1.ClusterRole{}
		err := s.c.Get(ctx, router.Key("", ruleName), rule)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, false, err
		}

		for _, policyRule := range rule.Rules {
			if slices.Contains(policyRule.APIGroups, "") &&
				slices.Contains(policyRule.Verbs, "list") &&
				slices.Contains(policyRule.Resources, "namespaces") &&
				len(policyRule.ResourceNames) == 0 {
				return nil, true, nil
			}

			if slices.Contains(policyRule.APIGroups, "") &&
				slices.Contains(policyRule.Verbs, "get") &&
				slices.Contains(policyRule.Resources, "namespaces") {
				result.Insert(policyRule.ResourceNames...)
			}
		}
	}

	return result, false, nil
}

func (s Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	names, all, err := s.allowed(ctx)
	if err != nil {
		return s.NewList(), err
	}

	list, err := s.next.List(ctx, namespace, opts)
	if err != nil {
		return s.NewList(), err
	}

	if all {
		return list, nil
	}

	var (
		items    = list.(*apiv1.ProjectList)
		filtered []apiv1.Project
	)

	for _, project := range items.Items {
		if names.Has(project.Name) {
			filtered = append(filtered, project)
		}
	}

	items.Items = filtered
	return items, nil
}

func (s Strategy) New() types.Object {
	return &apiv1.Project{}
}

func (s Strategy) NewList() types.ObjectList {
	return &apiv1.ProjectList{}
}
