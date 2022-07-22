package namespace

import (
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AssignNamespace(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)
	if appInstance.Status.Namespace != "" {
		return nil
	}

	appInstance.Status.Namespace = name.SafeConcatName(appInstance.Name, string(appInstance.UID)[:8])
	resp.Objects(appInstance)
	return nil
}

func CreateNamespace(req router.Request, resp router.Response) error {
	app := req.Object.(*v1.AppInstance)
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}
	addNamespace(cfg, app, resp)
	return nil
}

func addNamespace(cfg *apiv1.Config, appInstance *v1.AppInstance, resp router.Response) {
	labels := map[string]string{
		labels.AcornAppName:      appInstance.Name,
		labels.AcornAppNamespace: appInstance.Namespace,
		labels.AcornManaged:      "true",
	}

	if *cfg.SetPodSecurityEnforceProfile {
		labels["pod-security.kubernetes.io/enforce"] = cfg.PodSecurityEnforceProfile
	}

	resp.Objects(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   appInstance.Status.Namespace,
			Labels: labels,
		},
	})
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
