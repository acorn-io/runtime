package strategy

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type TableAdapter struct {
	strategy              any
	defaultTableConverter rest.TableConvertor
}

func NewTable(strategy any) *TableAdapter {
	return &TableAdapter{
		strategy:              strategy,
		defaultTableConverter: rest.NewDefaultTableConvertor(schema.GroupResource{}),
	}
}

func (t *TableAdapter) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if o, ok := t.strategy.(rest.TableConvertor); ok && o != nil {
		return o.ConvertToTable(ctx, object, tableOptions)
	}
	return t.defaultTableConverter.ConvertToTable(ctx, object, tableOptions)
}
