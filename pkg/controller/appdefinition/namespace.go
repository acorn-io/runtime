package appdefinition

import (
	"strings"

	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/condition"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func AssignNamespace(req router.Request, resp router.Response) (err error) {
	appInstance := req.Object.(*v1.AppInstance)
	cond := condition.Setter(appInstance, resp, v1.AppInstanceConditionNamespace)
	defer func() {
		cond.Error(err)
		// clear the error
		err = nil
	}()

	parts := strings.Split(appInstance.Name, ".")
	appInstance.Status.Namespace = name.SafeConcatName(parts[len(parts)-1], appInstance.ShortID())

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
		if req.Object == nil {
			return nil
		}
		appInstance := req.Object.(*v1.AppInstance)
		if appInstance.Status.Namespace == "" {
			return nil
		}
		return h.Handle(req, resp)
	})
}

func AddAcornProjectLabel(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	var projectNamespace corev1.Namespace

	if err := req.Client.Get(req.Ctx, kclient.ObjectKey{
		Name: app.Namespace,
	}, &projectNamespace); err != nil {
		return err
	}
	if projectNamespace.Labels == nil {
		projectNamespace.Labels = map[string]string{}
	}
	if projectNamespace.Labels[labels.AcornProject] != "true" {
		projectNamespace.Labels[labels.AcornProject] = "true"
		if err := req.Client.Update(req.Ctx, &projectNamespace); err != nil {
			return err
		}
	}
	return nil
}
