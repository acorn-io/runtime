package ref

import (
	"context"
	"fmt"
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/publicname"
	"github.com/acorn-io/runtime/pkg/scheme"
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

func typeString(obj kclient.Object) string {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return "unknown"
	}
	return strings.ToLower(strings.ReplaceAll(gvk.Kind, "Instance", ""))
}

func Lookup(ctx context.Context, req kclient.Client, out kclient.Object, namespace string, parts ...string) error {
	r := &resolver{
		ctx: ctx,
		req: req,
	}

	var (
		validSecrets *[]string
	)
	for i, name := range parts {
		if i+1 == len(parts) {
			switch v := out.(type) {
			case *corev1.Secret:
				return r.getSecret(v, namespace, name, validSecrets)
			case *v1.ServiceInstance:
				app, err := r.getAcorn(namespace, name)
				if apierrors.IsNotFound(err) {
					err = r.getService(v, namespace, name)
				} else if err == nil {
					err = r.getServiceForAcorn(v, app)
				}
				return err
			default:
				return fmt.Errorf("can not marshal expr [%s] to type %s", strings.Join(parts, "."), typeString(out))
			}
		}

		if validSecrets == nil {
			if v, ok := out.(*corev1.Secret); ok {
				// Support binding existing secrets with "." in the name, i.e. my-old-app.secret-name
				if err := r.getSecret(v, namespace, strings.Join(parts, "."), validSecrets); !apierrors.IsNotFound(err) {
					return err
				}
			}

			app, err := r.getAcorn(namespace, name)
			if apierrors.IsNotFound(err) {
				svc := &v1.ServiceInstance{}
				if err := r.getService(svc, namespace, name); err != nil {
					return err
				}
				validSecrets = &svc.Spec.Secrets
				namespace = svc.Namespace
			} else if err != nil {
				return err
			} else {
				validSecrets = nil
				namespace = app.Status.Namespace
			}
		} else if len(parts)-i != 2 || parts[i] != "secrets" && parts[i] != "secret" {
			// if validSecrets is set then we already found a service and we are evaluating the second
			// to last part which is invalid as it must be a secret name and the last part at this point
			return apierrors.NewNotFound(schema.GroupResource{
				Group:    corev1.SchemeGroupVersion.Group,
				Resource: "Secret",
			}, parts[i+1])
		}
	}

	return nil
}

func (r *resolver) getServiceForAcorn(out *v1.ServiceInstance, app *v1.AppInstance) error {
	svcs := &v1.ServiceInstanceList{}
	if app.Status.Namespace != "" {
		err := r.req.List(r.ctx, svcs, &kclient.ListOptions{
			Namespace: app.Status.Namespace,
		})
		if err != nil {
			return err
		}
	}
	for _, svc := range svcs.Items {
		if svc.Spec.Default {
			svc.DeepCopyInto(out)
			return nil
		}
	}

	return apierrors.NewNotFound(schema.GroupResource{
		Group:    v1.SchemeGroupVersion.Group,
		Resource: "ServiceInstance",
	}, app.Name)
}

func (r *resolver) getService(svc *v1.ServiceInstance, namespace, name string) error {
	if err := r.req.Get(r.ctx, router.Key(namespace, name), svc); err != nil {
		return err
	}

	if svc.Spec.External != "" {
		if publicname.Get(svc) == svc.Spec.External {
			return nil
		}
		return Lookup(r.ctx, r.req, svc, svc.Spec.AppNamespace, strings.Split(svc.Spec.External, ".")...)
	} else if svc.Spec.Alias != "" {
		if svc.Name == svc.Spec.Alias {
			return nil
		}
		return Lookup(r.ctx, r.req, svc, svc.Namespace, strings.Split(svc.Spec.Alias, ".")...)
	}

	return nil
}

func (r *resolver) getSecret(secret *corev1.Secret, namespace, name string, validSecrets *[]string) error {
	if validSecrets != nil {
		var found bool
		for _, secretName := range *validSecrets {
			if secretName == name {
				found = true
			}
		}
		if !found {
			return apierrors.NewNotFound(schema.GroupResource{
				Group:    v1.SchemeGroupVersion.Group,
				Resource: "Secret",
			}, name)
		}
	}
	if err := r.req.Get(r.ctx, router.Key(namespace, name), secret); err != nil {
		if apierrors.IsNotFound(err) {
			// Try finding by public name
			secretList := &corev1.SecretList{}
			if err := r.req.List(r.ctx, secretList, &kclient.ListOptions{
				LabelSelector: klabels.SelectorFromSet(map[string]string{
					labels.AcornPublicName: name,
				}),
				Namespace: namespace,
			}); err != nil {
				return err
			}

			if len(secretList.Items) == 1 {
				secretList.Items[0].DeepCopyInto(secret)
				return nil
			}
		}
		return err
	}

	return nil
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
