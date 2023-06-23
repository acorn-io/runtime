package publicname

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	"github.com/acorn-io/mink/pkg/types"
	"github.com/acorn-io/runtime/pkg/labels"
	"k8s.io/apimachinery/pkg/api/meta"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ translation.Translator = (*Translator)(nil)

type Translator struct {
	strategy strategy.CompleteStrategy
}

func (p *Translator) ListOpts(ctx context.Context, namespace string, opts storage.ListOptions) (string, storage.ListOptions, error) {
	return namespace, opts, nil
}

func (p *Translator) NewPublic() types.Object {
	return p.strategy.New()
}

func (p *Translator) NewPublicList() types.ObjectList {
	return p.strategy.NewList()
}

func NewStrategy(strategy strategy.CompleteStrategy) strategy.CompleteStrategy {
	return translation.NewTranslationStrategy(&Translator{
		strategy: strategy,
	}, strategy)
}

func (p *Translator) ToPublic(ctx context.Context, objs ...runtime.Object) (result []types.Object, _ error) {
	for _, obj := range objs {
		newObj := obj.DeepCopyObject().(types.Object)
		newObj.SetName(Get(newObj))
		result = append(result, newObj)
	}
	return result, nil
}

func (p *Translator) FromPublic(ctx context.Context, obj runtime.Object) (types.Object, error) {
	kobj := obj.DeepCopyObject().(kclient.Object)
	privateNamespace, privateName, err := p.FromPublicName(ctx, kobj.GetNamespace(), kobj.GetName())
	if err != nil {
		return nil, err
	}
	kobj.SetNamespace(privateNamespace)
	kobj.SetName(privateName)
	return kobj, nil
}

func (p *Translator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	parentName, _ := Split(name)
	if parentName == "" {
		return namespace, name, nil
	}
	list, err := p.strategy.List(ctx, namespace, storage.ListOptions{
		ResourceVersion:      "",
		ResourceVersionMatch: "",
		Predicate: storage.SelectionPredicate{
			Label: klabels.SelectorFromSet(klabels.Set{
				labels.AcornPublicName: name,
			}),
		},
	})
	if err != nil {
		return "", "", err
	}
	err = meta.EachListItem(list, func(object runtime.Object) error {
		name = object.(kclient.Object).GetName()
		return nil
	})
	return namespace, name, err
}
