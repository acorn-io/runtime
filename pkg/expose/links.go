package expose

import (
	"fmt"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Links(req router.Request, app *v1.AppInstance) (result []kclient.Object, _ error) {
	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return nil, err
	}

	for _, link := range app.Spec.Links {
		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      link.Target,
				Namespace: app.Status.Namespace,
				Labels: labels.Managed(app,
					labels.AcornLinkName, link.Service),
			},
			Spec: corev1.ServiceSpec{
				Type:         corev1.ServiceTypeExternalName,
				ExternalName: fmt.Sprintf("%s.%s.%s", link.Service, app.Namespace, cfg.InternalClusterDomain),
			},
		})
	}
	return
}
