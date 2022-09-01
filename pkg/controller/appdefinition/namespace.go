package appdefinition

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/condition"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func AssignNamespace(req router.Request, resp router.Response) (err error) {
	appInstance := req.Object.(*v1.AppInstance)
	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionNamespace)
	defer func() {
		cond.Error(err)
		// clear the error
		err = nil
	}()

	if appInstance.Spec.TargetNamespace != "" {
		if err := ensureNamespaceOwned(req, appInstance); err != nil {
			return err
		}
	}

	if appInstance.Spec.TargetNamespace == "" {
		appInstance.Status.Namespace = name.SafeConcatName(appInstance.Name, appInstance.ShortID())
	} else {
		appInstance.Status.Namespace = appInstance.Spec.TargetNamespace
	}

	resp.Objects(appInstance)
	return nil
}

func ensureNamespaceOwned(req router.Request, appInstance *v1.AppInstance) error {
	ns := &corev1.Namespace{}
	err := req.Get(ns, "", appInstance.Spec.TargetNamespace)
	if apierror.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}

	if ns.Labels[labels.AcornAppNamespace] != appInstance.Namespace ||
		ns.Labels[labels.AcornAppName] != appInstance.Name {
		return fmt.Errorf("can not use namespace %s, existing namespace must have labels [%s=%s] and [%s=%s]."+
			" And namespace will be deleted when the app is deleted",
			appInstance.Spec.TargetNamespace,
			ns.Labels[labels.AcornAppNamespace], appInstance.Namespace,
			ns.Labels[labels.AcornAppName], appInstance.Name)
	}
	return nil
}

func IgnoreTerminatingNamespace(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		ns := &corev1.Namespace{}
		if err := req.Get(ns, "", req.Namespace); err != nil {
			return err
		}
		if ns.Status.Phase == corev1.NamespaceTerminating {
			return nil
		}
		return h.Handle(req, resp)
	})
}

func RequireNamespace(h router.Handler) router.Handler {
	return router.HandlerFunc(func(req router.Request, resp router.Response) error {
		appInstance := req.Object.(*v1.AppInstance)
		if appInstance.Status.Namespace == "" {
			return nil
		}
		return h.Handle(req, resp)
	})
}
