package namespace

import (
	"github.com/acorn-io/baaah/pkg/router"
	v1 "github.com/acorn-io/runtime/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func AddNamespace(req router.Request, resp router.Response) error {
	appInstance := req.Object.(*v1.AppInstance)

	cfg, err := config.Get(req.Ctx, req.Client)
	if err != nil {
		return err
	}

	var projectNamespace corev1.Namespace
	if err := req.Client.Get(req.Ctx, client.ObjectKey{
		Name: appInstance.Namespace,
	}, &projectNamespace); err != nil {
		return err
	}

	labelMap := map[string]string{
		labels.AcornAppName:      appInstance.Name,
		labels.AcornAppNamespace: appInstance.Namespace,
		labels.AcornManaged:      "true",
	}

	labelMap = labels.Merge(labelMap, labels.GatherScoped("", "", appInstance.Status.AppSpec.Labels, nil, appInstance.Spec.Labels))
	annotations := labels.GatherScoped("", "", appInstance.Status.AppSpec.Annotations, nil, appInstance.Spec.Annotations)

	for _, key := range cfg.PropagateProjectAnnotations {
		if v, ok := projectNamespace.Annotations[key]; ok {
			annotations[key] = v
		}
	}

	for _, key := range cfg.PropagateProjectLabels {
		if v, ok := projectNamespace.Labels[key]; ok {
			labelMap[key] = v
		}
	}

	if *cfg.SetPodSecurityEnforceProfile {
		labelMap["pod-security.kubernetes.io/enforce"] = cfg.PodSecurityEnforceProfile
	}

	resp.Objects(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        appInstance.Status.Namespace,
			Labels:      labelMap,
			Annotations: annotations,
		},
	})
	return nil
}
