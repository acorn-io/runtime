package namespace

import (
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	v1 "github.com/ibuildthecloud/herd/pkg/apis/herd-project.io/v1"
	"github.com/ibuildthecloud/herd/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	apierror "k8s.io/apimachinery/pkg/api/errors"
)

func DeleteOrphaned(req router.Request, resp router.Response) error {
	ns := req.Object.(*corev1.Namespace)
	if ns.Status.Phase != corev1.NamespaceActive {
		return nil
	}

	appName := req.Object.GetLabels()[labels.HerdAppName]
	appNamespace := req.Object.GetLabels()[labels.HerdAppNamespace]

	err := req.Client.Get(&v1.AppInstance{}, appName, &meta.GetOptions{
		Namespace: appNamespace,
	})
	if apierror.IsNotFound(err) {
		return req.Client.Delete(ns)
	}
	return err

}
