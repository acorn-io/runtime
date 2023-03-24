package expr

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/scheme"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/baaah/pkg/typed"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type resolver struct {
	ctx context.Context
	req kclient.Client
}

func AssertType[T kclient.Object](obj kclient.Object, expr string) (result T, _ error) {
	if o, ok := obj.(T); ok {
		return o, nil
	}
	return result, fmt.Errorf("expression [%s] does not resolve to type [%s], got [%s]", expr, typeString(obj), typeString(typed.New[T]()))
}

func typeString(obj kclient.Object) string {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return "unknown"
	}
	return strings.ToLower(strings.ReplaceAll(gvk.Kind, "Instance", ""))
}

func Resolve(ctx context.Context, req kclient.Client, namespace, expr string) (kclient.Object, error) {
	r := &resolver{
		ctx: ctx,
		req: req,
	}

	var (
		parts        = strings.Split(strings.TrimSpace(expr), ".")
		obj          kclient.Object
		validSecrets *[]string
		err          error
	)
	for i, name := range parts {
		if validSecrets == nil {
			if i+1 == len(parts) {
				// if it's the last item it can't be an app
				obj, err = r.getService(namespace, name)
			} else {
				obj, err = r.getAcorn(namespace, name)
				if apierrors.IsNotFound(err) {
					obj, err = r.getService(namespace, name)
				}
			}
		}
		if validSecrets != nil || apierrors.IsNotFound(err) {
			obj, err = r.getSecret(namespace, name, validSecrets)
		}
		if err != nil {
			return nil, err
		}
		if app, ok := obj.(*v1.AppInstance); ok {
			namespace = app.Status.Namespace
		} else if svc, ok := obj.(*v1.ServiceInstance); ok {
			validSecrets = &svc.Spec.Secrets
			namespace = svc.Namespace
		} else if _, ok := obj.(*corev1.Secret); ok && i+1 != len(parts) {
			return nil, apierrors.NewNotFound(schema.GroupResource{
				Group:    corev1.SchemeGroupVersion.Group,
				Resource: "Secret",
			}, parts[i+1])
		}
	}

	if app, ok := obj.(*v1.AppInstance); ok {
		return r.getServiceForAcorn(app)
	}

	return obj, nil
}

func (r *resolver) getServiceForAcorn(app *v1.AppInstance) (*v1.ServiceInstance, error) {
	svcs := &v1.ServiceInstanceList{}
	if app.Status.Namespace != "" {
		err := r.req.List(r.ctx, svcs, &kclient.ListOptions{
			Namespace: app.Status.Namespace,
		})
		if err != nil {
			return nil, err
		}
	}
	for _, svc := range svcs.Items {
		if svc.Spec.Default {
			return &svc, nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "ServiceInstance",
	}, app.Name)
}

func (r *resolver) getService(namespace, name string) (*v1.ServiceInstance, error) {
	svc := &v1.ServiceInstance{}
	if err := r.req.Get(r.ctx, router.Key(namespace, name), svc); err != nil {
		return nil, err
	}

	if svc.Spec.External == "" {
		return svc, nil
	}

	ns := &corev1.Namespace{}
	err := r.req.Get(r.ctx, router.Key("", svc.Namespace), nil)
	if err != nil {
		return nil, err
	}

	obj, err := Resolve(r.ctx, r.req, ns.Labels[labels.AcornAppNamespace], svc.Spec.External)
	if err != nil {
		return nil, err
	}

	if svc, ok := obj.(*v1.ServiceInstance); ok {
		return svc, nil
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "AppInstance",
	}, name)
}

func (r *resolver) getSecret(namespace, name string, validSecrets *[]string) (*corev1.Secret, error) {
	if validSecrets != nil {
		var found bool
		for _, secretName := range *validSecrets {
			if secretName == name {
				found = true
			}
		}
		if !found {
			return nil, apierrors.NewNotFound(schema.GroupResource{
				Group:    v1.SchemeGroupVersion.Group,
				Resource: "Secret",
			}, name)
		}
	}
	svc := &corev1.Secret{}
	return svc, r.req.Get(r.ctx, router.Key(namespace, name), svc)
}

func (r *resolver) getAcorn(namespace, name string) (*v1.AppInstance, error) {
	// Two scenarios
	// 1. Looking up an app in a namespace created by an app, in this situation we are looking at
	//    app.spec and appspec to determine the acorn
	// 2. Looking up an app in a project namespace, in which we just look for an acorn by name

	ns := &corev1.Namespace{}
	err := r.req.Get(r.ctx, router.Key("", namespace), ns)
	if err != nil {
		return nil, err
	}

	parentAppName, parentAppNamespace := ns.Labels[labels.AcornAppName], ns.Labels[labels.AcornAppNamespace]
	if parentAppName == "" || parentAppNamespace == "" {
		app := &v1.AppInstance{}
		err := r.req.Get(r.ctx, router.Key(namespace, name), app)
		return app, err
	}

	parentApps := &v1.AppInstanceList{}
	err = r.req.List(r.ctx, parentApps, &kclient.ListOptions{
		LabelSelector: klabels.SelectorFromSet(map[string]string{
			labels.AcornAcornName:       name,
			labels.AcornParentAcornName: parentAppName,
		}),
		Namespace: parentAppNamespace,
	})
	if err != nil {
		return nil, err
	}

	if len(parentApps.Items) == 1 {
		return &parentApps.Items[0], nil
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "AppInstance",
	}, name)
}
