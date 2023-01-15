package projects

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/mink/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apiserver/pkg/storage"
)

type Translator struct {
}

func (t *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	return namespace, name, nil
}

func (t *Translator) ListOpts(namespace string, opts storage.ListOptions) (string, storage.ListOptions) {
	sel := opts.Predicate.Label
	if sel == nil {
		sel = klabels.Everything()
	}

	req, _ := klabels.NewRequirement(labels.AcornProject, selection.Equals, []string{"true"})
	sel = sel.Add(*req)

	opts.Predicate.Label = sel
	return namespace, opts
}

func (t *Translator) ToPublic(obj ...runtime.Object) (result []types.Object) {
	for _, obj := range obj {
		result = append(result, &apiv1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: obj.(*corev1.Namespace).Name,
			},
		})
	}
	return
}

func (t *Translator) FromPublic(ctx context.Context, obj runtime.Object) (types.Object, error) {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: obj.(*apiv1.Project).Name,
		},
	}, nil
}

func (t *Translator) NewPublic() types.Object {
	return &apiv1.Project{}
}

func (t *Translator) NewPublicList() types.ObjectList {
	return &apiv1.ProjectList{}
}
