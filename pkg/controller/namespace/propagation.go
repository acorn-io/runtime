package namespace

import (
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LabelsAnnotations(req router.Request, _ router.Response) error {
	projectNamespace := req.Object.(*corev1.Namespace)

	var appNamespaces corev1.NamespaceList
	if err := req.Client.List(req.Ctx, &appNamespaces, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			"acorn.io/app-namespace": projectNamespace.Name,
		}),
	}); err != nil {
		return err
	}

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	for _, appNs := range appNamespaces.Items {
		if appNs.Annotations == nil {
			appNs.Annotations = map[string]string{}
		}
		if appNs.Labels == nil {
			appNs.Labels = map[string]string{}
		}
		for _, key := range cfg.AnnotationsPropagation {
			if v, ok := projectNamespace.Annotations[key]; ok {
				appNs.Annotations[key] = v
			}
		}

		for _, key := range cfg.LabelsPropagation {
			if v, ok := projectNamespace.Labels[key]; ok {
				appNs.Labels[key] = v
			}
		}
		if err := req.Client.Update(req.Ctx, &appNs); err != nil {
			return err
		}
	}

	return nil
}
