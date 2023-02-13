package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
)

var _ rest.Lister = (*ListAdapter)(nil)

type GetAttr interface {
	GetAttr(obj runtime.Object) (labels.Set, fields.Set, error)
}

type GetToLister interface {
	GetToList(ctx context.Context, namespace, name string) (types.ObjectList, error)
}

type Lister interface {
	List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error)
	New() types.Object
	NewList() types.ObjectList
}

type ListAdapter struct {
	*TableAdapter
	strategy Lister
}

func NewList(strategy Lister) *ListAdapter {
	return &ListAdapter{
		TableAdapter: NewTable(strategy),
		strategy:     strategy,
	}
}

func (l *ListAdapter) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	label := labels.Everything()
	if options != nil && options.LabelSelector != nil {
		label = options.LabelSelector
	}
	field := fields.Everything()
	if options != nil && options.FieldSelector != nil {
		field = options.FieldSelector
	}
	out, err := l.ListPredicate(ctx, l.predicate(label, field), options)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (l *ListAdapter) NamespaceScoped() bool {
	if o, ok := l.strategy.(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	if o, ok := l.strategy.New().(NamespaceScoper); ok {
		return o.NamespaceScoped()
	}
	return true
}

func (l *ListAdapter) defaultGetAttr() storage.AttrFunc {
	return func(obj runtime.Object) (labels.Set, fields.Set, error) {
		labels := labels.Set{}
		fields := fields.Set{}

		var baseFunc storage.AttrFunc = storage.DefaultNamespaceScopedAttr
		if !l.NamespaceScoped() {
			baseFunc = storage.DefaultClusterScopedAttr
		}

		l, f, err := baseFunc(obj)
		if err != nil {
			return nil, nil, err
		}
		for k, v := range l {
			labels[k] = v
		}
		for k, v := range f {
			fields[k] = v
		}

		if f, ok := obj.(types.Fields); ok {
			for _, field := range f.FieldNames() {
				fields[field] = f.Get(field)
			}
		}

		return labels, fields, nil
	}
}

func (l *ListAdapter) predicate(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	result := storage.SelectionPredicate{
		Label: label,
		Field: field,
	}
	if attr, ok := l.strategy.(GetAttr); ok {
		result.GetAttrs = attr.GetAttr
	} else {
		result.GetAttrs = l.defaultGetAttr()
	}
	return result
}

// ListPredicate returns a list of all the items matching the given
// SelectionPredicate.
func (l *ListAdapter) ListPredicate(ctx context.Context, p storage.SelectionPredicate, options *metainternalversion.ListOptions) (runtime.Object, error) {
	if options == nil {
		// By default we should serve the request from etcd.
		options = &metainternalversion.ListOptions{ResourceVersion: ""}
	}
	p.Limit = options.Limit
	p.Continue = options.Continue
	ns, _ := request.NamespaceFrom(ctx)
	storageOpts := storage.ListOptions{
		ResourceVersion:      options.ResourceVersion,
		ResourceVersionMatch: options.ResourceVersionMatch,
		Predicate:            p,
	}
	if name, ok := p.MatchesSingle(); ok {
		if gtl, ok := l.strategy.(GetToLister); ok {
			return gtl.GetToList(ctx, ns, name)
		}
	}

	return l.strategy.List(ctx, ns, storageOpts)
}

func (l *ListAdapter) NewList() runtime.Object {
	return l.strategy.NewList()
}
