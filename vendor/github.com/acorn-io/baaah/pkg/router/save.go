package router

import (
	"reflect"

	"github.com/acorn-io/baaah/pkg/apply"
	"github.com/acorn-io/baaah/pkg/backend"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type save struct {
	apply  apply.Apply
	cache  backend.CacheFactory
	client kclient.Client
}

func (s *save) save(unmodified runtime.Object, req Request, resp *response, watchingGVKS []schema.GroupVersionKind) (kclient.Object, error) {
	var owner = req.Object
	if owner == nil {
		owner := &unstructured.Unstructured{}
		owner.SetGroupVersionKind(req.GVK)
		owner.SetNamespace(req.Namespace)
		owner.SetName(req.Name)
	}
	apply := s.apply.
		WithPruneGVKs(watchingGVKS...)

	// Special case the situation where there are no objects and a retry later is set.
	// In this situation don't purge all the objects previously created
	if resp.noPrune || len(resp.objects) == 0 && resp.delay > 0 {
		apply = apply.WithNoPrune()
	}
	if err := apply.Apply(req.Ctx, owner, resp.objects...); err != nil {
		return nil, err
	}

	newObj := req.Object
	if newObj != nil && StatusChanged(unmodified, newObj) {
		return newObj, s.client.Status().Update(req.Ctx, newObj)
	}

	return newObj, nil
}

func statusField(obj runtime.Object) interface{} {
	v := reflect.ValueOf(obj).Elem()
	fieldValue := v.FieldByName("Status")
	if fieldValue.Kind() == reflect.Invalid {
		return nil
	}
	return fieldValue.Interface()
}

func StatusChanged(unmodified, newObj runtime.Object) bool {
	return !equality.Semantic.DeepEqual(statusField(unmodified), statusField(newObj))
}
