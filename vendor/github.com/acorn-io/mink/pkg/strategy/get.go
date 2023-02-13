package strategy

import (
	"context"

	"github.com/acorn-io/mink/pkg/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
)

var _ rest.Getter = (*GetAdapter)(nil)

type Getter interface {
	Get(ctx context.Context, namespace, name string) (types.Object, error)
}

func NewGet(strategy Getter) *GetAdapter {
	return &GetAdapter{
		strategy: strategy,
	}
}

type GetAdapter struct {
	strategy Getter
}

func (a *GetAdapter) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)
	return a.strategy.Get(ctx, ns, name)
}
