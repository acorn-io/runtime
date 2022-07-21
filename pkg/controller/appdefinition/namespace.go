package appdefinition

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
)

func AssignNamespace(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace != "" {
		return nil
	}

	appInstance.Status.Namespace = name.SafeConcatName(appInstance.Name, appInstance.ShortID())
	resp.Objects(appInstance)
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
