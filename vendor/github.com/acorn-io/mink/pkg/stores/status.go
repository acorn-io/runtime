package stores

import (
	"context"

	"github.com/acorn-io/mink/pkg/strategy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type Status struct {
	update                *strategy.UpdateAdapter
	get                   *strategy.GetAdapter
	strategy              any
	defaultTableConverter rest.TableConvertor
}

func NewStatus(scheme *runtime.Scheme, updater strategy.StatusUpdater) rest.Storage {
	return &Status{
		update:                strategy.NewUpdateStatus(scheme, updater),
		get:                   strategy.NewGet(updater),
		strategy:              updater,
		defaultTableConverter: rest.NewDefaultTableConvertor(schema.GroupResource{}),
	}
}

func (s *Status) New() runtime.Object {
	return s.update.New()
}

func (s *Status) Destroy() {
}

func (s *Status) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return s.get.Get(ctx, name, options)
}

func (s *Status) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return s.update.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

func (s *Status) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if o, ok := s.strategy.(rest.TableConvertor); ok {
		return o.ConvertToTable(ctx, object, tableOptions)
	}
	return s.defaultTableConverter.ConvertToTable(ctx, object, tableOptions)
}
