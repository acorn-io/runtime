package uncached

import (
	"k8s.io/apimachinery/pkg/runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func List(obj kclient.ObjectList) kclient.ObjectList {
	return &HolderList{
		ObjectList: obj,
	}
}

func Get(obj kclient.Object) kclient.Object {
	return &Holder{
		Object: obj,
	}
}

func IsWrapped(obj runtime.Object) bool {
	if _, ok := obj.(*Holder); ok {
		return true
	}
	if _, ok := obj.(*HolderList); ok {
		return true
	}
	return false
}

func Unwrap(obj runtime.Object) runtime.Object {
	if h, ok := obj.(*Holder); ok {
		return h.Object
	}
	if h, ok := obj.(*HolderList); ok {
		return h.ObjectList
	}
	return obj
}

func UnwrapList(obj kclient.ObjectList) kclient.ObjectList {
	if h, ok := obj.(*HolderList); ok {
		return h.ObjectList
	}
	return obj
}

type Holder struct {
	kclient.Object
}

func (h *Holder) DeepCopyObject() runtime.Object {
	return &Holder{Object: h.Object.DeepCopyObject().(kclient.Object)}
}

type HolderList struct {
	kclient.ObjectList
}

func (h *HolderList) DeepCopyObject() runtime.Object {
	return &HolderList{ObjectList: h.ObjectList.DeepCopyObject().(kclient.ObjectList)}
}
