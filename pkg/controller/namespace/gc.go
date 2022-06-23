package namespace

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
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

	err := req.Client.Get(req.Ctx, router.Key(appNamespace, appName), &v1.AppInstance{})
	if apierror.IsNotFound(err) {
		return req.Client.Delete(req.Ctx, ns)
	}
	return err
}
