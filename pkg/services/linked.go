package services

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func defaultService(ctx context.Context, c kclient.Client, appInstance *v1.AppInstance) (*v1.ServiceInstance, error) {
	if appInstance.Status.Namespace == "" {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    v1.SchemeGroupVersion.Group,
			Resource: "ServiceInstance",
		}, "app namespace")
	}

	services := &v1.ServiceInstanceList{}
	err := c.List(ctx, services, &kclient.ListOptions{
		Namespace: appInstance.Status.Namespace,
	})
	if err != nil {
		return nil, err
	}
	var def *v1.ServiceInstance
	for _, svc := range services.Items {
		if svc.Spec.Default {
			if def != nil {
				return nil, fmt.Errorf("two default services found for app [%s/%s]", appInstance.Namespace, appInstance.Name)
			}
			copy := svc
			def = &copy
		}
	}

	if def == nil {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    v1.SchemeGroupVersion.Group,
			Resource: "ServiceInstance",
		}, "default")
	}

	return def, nil
}

func resolveTargetService(ctx context.Context, c kclient.Client, name, namespace string) (*corev1.Service, error) {
	nameParts := strings.Split(name, ".")

	for i := range nameParts {
		if len(nameParts) == i+1 {
			acornSvc := &v1.ServiceInstance{}
			if err := c.Get(ctx, router.Key(namespace, name), acornSvc); apierrors.IsNotFound(err) {
				app := &v1.AppInstance{}
				if err := c.Get(ctx, router.Key(namespace, name), app); err != nil {
					return nil, err
				}
				acornSvc, err = defaultService(ctx, c, app)
				if err != nil {
					return nil, err
				}
				namespace = acornSvc.Namespace
				name = acornSvc.Name
			} else if err != nil {
				return nil, err
			}
			svc := &corev1.Service{}
			return svc, c.Get(ctx, router.Key(namespace, name), svc)
		}

		app := &v1.AppInstance{}
		err := c.Get(ctx, router.Key(namespace, name), app)
		if err != nil {
			return nil, err
		}
		name = nameParts[i+1]
		namespace = app.Status.Namespace
	}

	panic("bug: unreachable line")
}
