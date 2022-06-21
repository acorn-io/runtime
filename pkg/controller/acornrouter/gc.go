package acornrouter

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/rancher/wrangler/pkg/apply"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const ownerGVK = "acorn.io/v1, Kind=AppInstance"

func GCAcornRouter(req router.Request, resp router.Response) error {
	ds := req.Object.(*appsv1.DaemonSet)
	ownerType := ds.Annotations[apply.LabelGVK]
	name := ds.Annotations[apply.LabelName]
	namespace := ds.Annotations[apply.LabelNamespace]

	if ownerType != ownerGVK {
		return nil
	}

	var app v1.AppInstance
	err := req.Get(&app, namespace, name)
	if apierrors.IsNotFound(err) {
		err := req.Client.Delete(req.Ctx, ds)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return err
}

func GCAcornRouterService(req router.Request, resp router.Response) error {
	ds := req.Object.(*corev1.Service)
	ownerType := ds.Annotations[apply.LabelGVK]
	name := ds.Annotations[apply.LabelName]
	namespace := ds.Annotations[apply.LabelNamespace]

	if ownerType != ownerGVK {
		return nil
	}

	var app v1.AppInstance
	err := req.Get(&app, namespace, name)
	if apierrors.IsNotFound(err) {
		err := req.Client.Delete(req.Ctx, ds)
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return err
}
