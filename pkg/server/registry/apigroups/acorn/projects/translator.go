package projects

import (
	"context"
	"strings"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/mink/pkg/types"
	corev1 "k8s.io/api/core/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/storage"
)

type Translator struct {
	defaultRegion string
}

func (t *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	return namespace, name, nil
}

func (t *Translator) ListOpts(ctx context.Context, namespace string, opts storage.ListOptions) (string, storage.ListOptions, error) {
	sel := opts.Predicate.Label
	if sel == nil {
		sel = klabels.Everything()
	}

	req, _ := klabels.NewRequirement(labels.AcornProject, selection.Equals, []string{"true"})
	sel = sel.Add(*req)

	opts.Predicate.Label = sel
	return namespace, opts, nil
}

func (t *Translator) ToPublic(ctx context.Context, obj ...runtime.Object) (result []types.Object, _ error) {
	for _, obj := range obj {
		ns := obj.(*corev1.Namespace)
		if !ns.DeletionTimestamp.IsZero() {
			continue
		}

		var supportedRegions []string
		if len(ns.Annotations[labels.AcornProjectSupportedRegions]) > 0 {
			supportedRegions = strings.Split(ns.Annotations[labels.AcornProjectSupportedRegions], ",")
		}

		defaultRegion := ns.Annotations[labels.AcornProjectDefaultRegion]

		calculatedDefaultRegion := ns.Annotations[labels.AcornCalculatedProjectDefaultRegion]
		if defaultRegion == "" && calculatedDefaultRegion == "" {
			calculatedDefaultRegion = t.defaultRegion
		}

		delete(ns.Labels, labels.AcornProject)
		delete(ns.Annotations, labels.AcornProjectDefaultRegion)
		delete(ns.Annotations, labels.AcornProjectSupportedRegions)

		result = append(result, &apiv1.Project{
			ObjectMeta: ns.ObjectMeta,
			Spec: apiv1.ProjectSpec{
				DefaultRegion:    defaultRegion,
				SupportedRegions: supportedRegions,
			},
			Status: apiv1.ProjectStatus{
				Namespace:     ns.Name,
				DefaultRegion: calculatedDefaultRegion,
			},
		})
	}
	return
}

func (t *Translator) FromPublic(ctx context.Context, obj runtime.Object) (types.Object, error) {
	prj := obj.(*apiv1.Project)

	ns := &corev1.Namespace{
		ObjectMeta: prj.ObjectMeta,
	}

	if ns.Labels == nil {
		ns.Labels = make(map[string]string)
	}
	ns.Labels[labels.AcornProject] = "true"

	if ns.Annotations == nil {
		ns.Annotations = make(map[string]string)
	}
	ns.Annotations[labels.AcornProjectDefaultRegion] = prj.Spec.DefaultRegion
	ns.Annotations[labels.AcornProjectSupportedRegions] = strings.Join(prj.Spec.SupportedRegions, ",")
	ns.Annotations[labels.AcornCalculatedProjectDefaultRegion] = prj.Status.DefaultRegion

	return ns, nil
}

func (t *Translator) NewPublic() types.Object {
	return &apiv1.Project{}
}

func (t *Translator) NewPublicList() types.ObjectList {
	return &apiv1.ProjectList{}
}
