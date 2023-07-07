package namespace

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func DeleteOrphaned(req router.Request, _ router.Response) error {
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

func DeleteProjectOnNamespaceDelete(req router.Request, _ router.Response) error {
	if req.Object != nil {
		return nil
	}

	err := req.Client.Delete(req.Ctx, &v1.ProjectInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	})
	if apierror.IsNotFound(err) {
		return nil
	}

	return err
}
