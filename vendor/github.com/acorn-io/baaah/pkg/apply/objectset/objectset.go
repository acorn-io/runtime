package objectset

import (
	"fmt"
	"reflect"
	"sort"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

type ObjectKey struct {
	Name      string
	Namespace string
}

func (o ObjectKey) String() string {
	if o.Namespace == "" {
		return o.Name
	}
	return fmt.Sprintf("%s/%s", o.Namespace, o.Name)
}

type ObjectKeyByGVK map[schema.GroupVersionKind][]ObjectKey

type ObjectByGVK map[schema.GroupVersionKind]map[ObjectKey]kclient.Object

func (o ObjectByGVK) Add(gvk schema.GroupVersionKind, obj kclient.Object) {
	objs := o[gvk]
	if objs == nil {
		objs = ObjectByKey{}
		o[gvk] = objs
	}

	objs[ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}] = obj
}

type ObjectSet struct {
	scheme      *runtime.Scheme
	objects     ObjectByGVK
	objectsByGK ObjectByGK
	order       []kclient.Object
	gvkOrder    []schema.GroupVersionKind
	gvkSeen     map[schema.GroupVersionKind]bool
}

func NewObjectSet(scheme *runtime.Scheme, objs ...kclient.Object) (*ObjectSet, error) {
	os := &ObjectSet{
		scheme:      scheme,
		objects:     ObjectByGVK{},
		objectsByGK: ObjectByGK{},
		gvkSeen:     map[schema.GroupVersionKind]bool{},
	}
	return os, os.Add(objs...)
}

func (o *ObjectSet) ObjectsByGVK() ObjectByGVK {
	if o == nil {
		return nil
	}
	return o.objects
}

func (o *ObjectSet) Contains(gk schema.GroupKind, key ObjectKey) bool {
	_, ok := o.objectsByGK[gk][key]
	return ok
}

func (o *ObjectSet) All() []kclient.Object {
	return o.order
}

func (o *ObjectSet) Add(objs ...kclient.Object) error {
	for _, obj := range objs {
		if err := o.add(obj); err != nil {
			return err
		}
	}
	return nil
}

func (o *ObjectSet) add(obj kclient.Object) error {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return nil
	}

	gvk, err := apiutil.GVKForObject(obj, o.scheme)
	if err != nil {
		return err
	}

	o.objects.Add(gvk, obj)
	o.objectsByGK.Add(gvk, obj)

	o.order = append(o.order, obj)
	if !o.gvkSeen[gvk] {
		o.gvkSeen[gvk] = true
		o.gvkOrder = append(o.gvkOrder, gvk)
	}

	return nil
}

func (o *ObjectSet) Len() int {
	return len(o.objects)
}

func (o *ObjectSet) GVKs() []schema.GroupVersionKind {
	return o.GVKOrder()
}

func (o *ObjectSet) GVKOrder(known ...schema.GroupVersionKind) []schema.GroupVersionKind {
	var rest []schema.GroupVersionKind

	for _, gvk := range known {
		if o.gvkSeen[gvk] {
			continue
		}
		rest = append(rest, gvk)
	}

	sort.Slice(rest, func(i, j int) bool {
		return rest[i].String() < rest[j].String()
	})

	return append(o.gvkOrder, rest...)
}

// Namespaces all distinct namespaces found on the objects in this set.
func (o *ObjectSet) Namespaces() []string {
	namespaces := sets.String{}
	for _, objsByKey := range o.ObjectsByGVK() {
		for objKey := range objsByKey {
			namespaces.Insert(objKey.Namespace)
		}
	}
	return namespaces.List()
}

type ObjectByKey map[ObjectKey]kclient.Object

func (o ObjectByKey) Namespaces() []string {
	namespaces := sets.String{}
	for objKey := range o {
		namespaces.Insert(objKey.Namespace)
	}
	return namespaces.List()
}

type ObjectByGK map[schema.GroupKind]map[ObjectKey]kclient.Object

func (o ObjectByGK) Add(gvk schema.GroupVersionKind, obj kclient.Object) {
	gk := gvk.GroupKind()

	objs := o[gk]
	if objs == nil {
		objs = ObjectByKey{}
		o[gk] = objs
	}

	objs[ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}] = obj
}
