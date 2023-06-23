package projects

import (
	"context"
	"strings"

	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/labels"
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

func (t *Translator) ToPublic(_ context.Context, obj ...runtime.Object) (result []types.Object, _ error) {
	for _, obj := range obj {
		ns := obj.(*corev1.Namespace)
		if !ns.DeletionTimestamp.IsZero() {
			continue
		}

		defaultRegion := ns.Annotations[labels.AcornProjectDefaultRegion]

		calculatedDefaultRegion := ns.Annotations[labels.AcornCalculatedProjectDefaultRegion]
		if calculatedDefaultRegion == "" {
			if defaultRegion == "" && len(ns.Annotations[labels.AcornProjectSupportedRegions]) == 0 {
				calculatedDefaultRegion = t.defaultRegion
			} else {
				calculatedDefaultRegion = defaultRegion
			}
		}

		var supportedRegions, calculatedSupportedRegions []string
		if len(ns.Annotations[labels.AcornProjectSupportedRegions]) > 0 {
			supportedRegions = strings.Split(ns.Annotations[labels.AcornProjectSupportedRegions], ",")
			calculatedSupportedRegions = supportedRegions
		} else {
			calculatedSupportedRegions = []string{calculatedDefaultRegion}
		}

		delete(ns.Labels, labels.AcornProject)
		delete(ns.Annotations, labels.AcornProjectDefaultRegion)
		delete(ns.Annotations, labels.AcornProjectSupportedRegions)
		delete(ns.Annotations, labels.AcornCalculatedProjectDefaultRegion)
		delete(ns.Annotations, labels.AcornCalculatedProjectSupportedRegions)

		result = append(result, &apiv1.Project{
			ObjectMeta: ns.ObjectMeta,
			Spec: apiv1.ProjectSpec{
				DefaultRegion:    defaultRegion,
				SupportedRegions: supportedRegions,
			},
			Status: apiv1.ProjectStatus{
				Namespace:        ns.Name,
				DefaultRegion:    calculatedDefaultRegion,
				SupportedRegions: calculatedSupportedRegions,
			},
		})
	}
	return
}

func (t *Translator) FromPublic(_ context.Context, obj runtime.Object) (types.Object, error) {
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
	ns.Annotations[labels.AcornCalculatedProjectSupportedRegions] = strings.Join(prj.Status.SupportedRegions, ",")

	return ns, nil
}

func (t *Translator) NewPublic() types.Object {
	return &apiv1.Project{}
}

func (t *Translator) NewPublicList() types.ObjectList {
	return &apiv1.ProjectList{}
}
