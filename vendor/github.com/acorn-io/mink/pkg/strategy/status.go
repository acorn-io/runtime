package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

var _ rest.Storage = (*Status)(nil)

type StatusUpdater interface {
	Getter
	Creater

	UpdateStatus(ctx context.Context, obj types.Object) (types.Object, error)
}

type Status struct {
	update                *UpdateAdapter
	get                   *GetAdapter
	strategy              any
	defaultTableConverter rest.TableConvertor
}

func NewStatus(scheme *runtime.Scheme, strategy StatusUpdater) *Status {
	return &Status{
		update: &UpdateAdapter{
			CreateAdapter: NewCreate(scheme, strategy),
			strategy:      strategy,
		},
		get:                   NewGet(strategy),
		strategy:              strategy,
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
	return s.update.update(ctx, true, name, objInfo, createValidation, updateValidation, false, options)
}

func (s *Status) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if o, ok := s.strategy.(rest.TableConvertor); ok {
		return o.ConvertToTable(ctx, object, tableOptions)
	}
	return s.defaultTableConverter.ConvertToTable(ctx, object, tableOptions)
}
