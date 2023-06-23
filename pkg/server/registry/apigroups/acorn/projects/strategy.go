package projects

import (
	"context"
	"fmt"
	"net/http"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/utils/strings/slices"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	c       kclient.Client
	creater strategy.Creater
	updater strategy.Updater
	lister  strategy.Lister
	deleter strategy.Deleter
}

func (s *Strategy) Create(ctx context.Context, object types.Object) (types.Object, error) {
	project := object.(*apiv1.Project)

	result, err := s.creater.Create(ctx, object)
	if !apierrors.IsAlreadyExists(err) {
		return result, err
	}

	ns := &corev1.Namespace{}
	getErr := s.c.Get(ctx, router.Key("", project.Name), ns)
	if getErr == nil {
		// Project is just a labeled namespace
		if ns.Labels[labels.AcornProject] != "true" {
			qualifiedResource := schema.GroupResource{
				Resource: "namespaces",
			}
			return nil, &apierrors.StatusError{
				ErrStatus: metav1.Status{
					Status: metav1.StatusFailure,
					Code:   http.StatusConflict,
					Reason: metav1.StatusReasonAlreadyExists,
					Details: &metav1.StatusDetails{
						Group: qualifiedResource.Group,
						Kind:  qualifiedResource.Resource,
						Name:  project.Name,
					},
					Message: fmt.Sprintf("%s %q already exists but does not contain the %s=true label",
						qualifiedResource.String(), project.Name, labels.AcornProject),
				},
			}
		}
	}
	return result, err
}

func (s *Strategy) Update(ctx context.Context, object types.Object) (types.Object, error) {
	return s.updater.Update(ctx, object)
}

// Get is based on list because list will do the RBAC checks to ensure the user can access that
// project. This also ensure that you can only delete a project that you have namespace access to
func (s *Strategy) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	list, err := s.List(ctx, namespace, storage.ListOptions{})
	if err != nil {
		return nil, err
	}
	projects := list.(*apiv1.ProjectList)
	for _, project := range projects.Items {
		if project.Name == name {
			return &project, nil
		}
	}
	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    apiv1.SchemeGroupVersion.Group,
		Resource: "projects",
	}, name)
}

func (s *Strategy) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	return s.deleter.Delete(ctx, obj)
}

func (s *Strategy) allowed(ctx context.Context) (sets.Set[string], bool, error) {
	user, ok := request.UserFrom(ctx)
	if !ok {
		return nil, false, nil
	}

	crbs := &rbacv1.ClusterRoleBindingList{}
	err := s.c.List(ctx, crbs)
	if err != nil {
		return nil, false, err
	}

	result := sets.New[string]()

	rulesName := sets.New[string]()
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
			case "ServiceAccount":
				if fmt.Sprintf("system:serviceaccount:%s:%s", subject.Namespace, subject.Name) == user.GetName() {
					rulesName.Insert(crb.RoleRef.Name)
				}
			}
		}
	}

	for _, ruleName := range sets.List(rulesName) {
		rule := &rbacv1.ClusterRole{}
		err := s.c.Get(ctx, router.Key("", ruleName), rule)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, false, err
		}

		for _, policyRule := range rule.Rules {
			if matches(policyRule.APIGroups, "") &&
				matches(policyRule.Verbs, "list") &&
				matches(policyRule.Resources, "namespaces") &&
				len(policyRule.ResourceNames) == 0 {
				return nil, true, nil
			}

			if matches(policyRule.APIGroups, "") &&
				matches(policyRule.Verbs, "get") &&
				matches(policyRule.Resources, "namespaces") {
				result.Insert(policyRule.ResourceNames...)
			}
		}
	}

	return result, false, nil
}

func matches(allowed []string, requested string) bool {
	return slices.Contains(allowed, "*") ||
		slices.Contains(allowed, requested)
}

func (s *Strategy) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	names, all, err := s.allowed(ctx)
	if err != nil {
		return s.NewList(), err
	}

	list, err := s.lister.List(ctx, namespace, opts)
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

func (s *Strategy) New() types.Object {
	return &apiv1.Project{}
}

func (s *Strategy) NewList() types.ObjectList {
	return &apiv1.ProjectList{}
}
