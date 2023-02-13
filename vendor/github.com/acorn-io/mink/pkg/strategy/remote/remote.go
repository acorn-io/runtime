package remote

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/storage"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ strategy.CompleteStrategy = (*Remote)(nil)

type Remote struct {
	obj     types.Object
	objList types.ObjectList
	c       kclient.WithWatch
}

func NewRemote(obj types.Object, objList types.ObjectList, c kclient.WithWatch) *Remote {
	return &Remote{
		obj:     obj,
		objList: objList,
		c:       c,
	}
}

func (r *Remote) Create(ctx context.Context, object types.Object) (types.Object, error) {
	return object, r.c.Create(ctx, object)
}

func (r *Remote) New() types.Object {
	return r.obj.DeepCopyObject().(types.Object)
}

func (r *Remote) Get(ctx context.Context, namespace, name string) (types.Object, error) {
	obj := r.New().(types.Object)
	return obj, r.c.Get(ctx, kclient.ObjectKey{Namespace: namespace, Name: name}, obj)
}

func (r *Remote) Update(ctx context.Context, obj types.Object) (types.Object, error) {
	return obj, r.c.Update(ctx, obj)
}

func (r *Remote) UpdateStatus(ctx context.Context, obj types.Object) (types.Object, error) {
	return obj, r.c.Status().Update(ctx, obj)
}

func (r *Remote) GetToList(ctx context.Context, namespace, name string) (types.ObjectList, error) {
	list := r.NewList().(types.ObjectList)
	return list, r.c.List(ctx, list, &kclient.ListOptions{
		FieldSelector: fields.SelectorFromSet(map[string]string{
			"metadata.name":      name,
			"metadata.namespace": namespace,
		}),
		Limit: 1,
	})
}

func (r *Remote) List(ctx context.Context, namespace string, opts storage.ListOptions) (types.ObjectList, error) {
	list := r.NewList().(types.ObjectList)
	return list, r.c.List(ctx, list, strategy.ToListOpts(namespace, opts))
}

func (r *Remote) NewList() types.ObjectList {
	return r.objList.DeepCopyObject().(types.ObjectList)
}

func (r *Remote) Delete(ctx context.Context, obj types.Object) (types.Object, error) {
	return obj, r.c.Delete(ctx, obj)
}

func (r *Remote) Watch(ctx context.Context, namespace string, opts storage.ListOptions) (<-chan watch.Event, error) {
	list := r.NewList().(types.ObjectList)
	listOpts := strategy.ToListOpts(namespace, opts)
	w, err := r.c.Watch(ctx, list, listOpts)
	if err != nil {
		return nil, err
	}
	return w.ResultChan(), nil
}

func (r *Remote) Destroy() {
}
