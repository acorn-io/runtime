package translation

import (
	"context"

	mtypes "github.com/acorn-io/mink/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
)

type SimpleTranslator interface {
	FromPublic(obj mtypes.Object) mtypes.Object
	ToPublic(obj mtypes.Object) mtypes.Object
}

func NewSimpleTranslator(translator SimpleTranslator, pubType mtypes.Object, pubTypeList mtypes.ObjectList) Translator {
	return &simpleTranslator{
		obj:        pubType,
		objList:    pubTypeList,
		translator: translator,
	}
}

type simpleTranslator struct {
	obj        mtypes.Object
	objList    mtypes.ObjectList
	translator SimpleTranslator
}

func (s *simpleTranslator) FromPublicName(ctx context.Context, namespace, name string) (string, string, error) {
	return namespace, name, nil
}

func (s *simpleTranslator) ListOpts(ctx context.Context, namespace string, opts storage.ListOptions) (string, storage.ListOptions, error) {
	return namespace, opts, nil
}

func (s *simpleTranslator) NewPublic() mtypes.Object {
	return s.obj.DeepCopyObject().(mtypes.Object)
}

func (s *simpleTranslator) NewPublicList() mtypes.ObjectList {
	return s.objList.DeepCopyObject().(mtypes.ObjectList)
}

func (s *simpleTranslator) FromPublic(ctx context.Context, obj runtime.Object) (result mtypes.Object, _ error) {
	return s.translator.FromPublic(obj.(mtypes.Object)), nil
}

func (s *simpleTranslator) ToPublic(ctx context.Context, objs ...runtime.Object) (result []mtypes.Object, _ error) {
	result = make([]mtypes.Object, 0, len(objs))
	for _, obj := range objs {
		result = append(result, s.translator.ToPublic(obj.(mtypes.Object)))
	}
	return
}
