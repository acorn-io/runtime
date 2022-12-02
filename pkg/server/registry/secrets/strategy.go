package secrets

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor
}

func NewStrategy(c kclient.WithWatch, expose bool) strategy.CompleteStrategy {
	storage := newStorage(c, expose)
	return NewStrategyWithStorage(c, storage)

}

func NewStrategyWithStorage(c kclient.WithWatch, storage strategy.CompleteStrategy) strategy.CompleteStrategy {
	return &Strategy{
		CompleteStrategy: storage,
		TableConvertor:   tables.SecretConverter,
	}
}

func newStorage(c kclient.WithWatch, expose bool) strategy.CompleteStrategy {
	return translation.NewTranslationStrategy(&Translator{
		c:      c,
		expose: expose,
	}, remote.NewRemote(&corev1.Secret{}, &corev1.SecretList{}, c))
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	sec := obj.(*apiv1.Secret)
	if sec.Type != "" {
		if !v1.SecretTypes[corev1.SecretType(v1.SecretTypePrefix+sec.Type)] {
			result = append(result, field.Invalid(field.NewPath("type"), sec.Type, "Invalid secret type"))
		}
	}
	return
}

func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return s.Validate(ctx, obj)
}
