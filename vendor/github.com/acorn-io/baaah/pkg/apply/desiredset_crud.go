package apply

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *apply) create(gvk schema.GroupVersionKind, obj kclient.Object) (kclient.Object, error) {
	a.log("creating", gvk, obj)
	return obj, a.client.Create(a.ctx, obj)
}

func (a *apply) get(gvk schema.GroupVersionKind, obj kclient.Object, namespace, name string) (kclient.Object, error) {
	if obj == nil {
		newObj, err := a.newObj(gvk, false)
		if err != nil {
			return nil, err
		}
		obj = newObj.(kclient.Object)
	} else {
		obj = obj.DeepCopyObject().(kclient.Object)
	}

	return obj, a.client.Get(a.ctx, kclient.ObjectKey{Namespace: namespace, Name: name}, obj)
}

func (a *apply) delete(gvk schema.GroupVersionKind, namespace, name string) error {
	ustr := &unstructured.Unstructured{}
	ustr.SetGroupVersionKind(gvk)
	ustr.SetName(name)
	ustr.SetNamespace(namespace)
	a.log("deleting", gvk, ustr)
	return a.client.Delete(a.ctx, ustr)
}
