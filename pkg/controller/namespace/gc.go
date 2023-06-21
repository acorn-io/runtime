package namespace

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func DeleteOrphaned(req router.Request, resp router.Response) error {
	ns := req.Object.(*corev1.Namespace)
	if ns.Status.Phase != corev1.NamespaceActive {
		return nil
	}

	appName := req.Object.GetLabels()[labels.AcornAppName]
	appNamespace := req.Object.GetLabels()[labels.AcornAppNamespace]

	if appName == "" || appNamespace == "" {
		return nil
	}

	err := req.Client.Get(req.Ctx, router.Key(appNamespace, appName), &v1.AppInstance{})
	if apierror.IsNotFound(err) {
		return req.Client.Delete(req.Ctx, ns)
	}
	return err
}
