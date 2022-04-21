package namespace

import (
	v1 "github.com/acorn-io/acorn/pkg/apis/acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/meta"
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

	err := req.Client.Get(&v1.AppInstance{}, appName, &meta.GetOptions{
		Namespace: appNamespace,
	})
	if apierror.IsNotFound(err) {
		return req.Client.Delete(ns)
	}
	return err

}
