package credentials

import (
	"context"
	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
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

func NewStrategy(c kclient.WithWatch, expose bool) *Strategy {
	return &Strategy{
		CompleteStrategy: translation.NewTranslationStrategy(&Translator{
			client: c,
			expose: expose,
		}, remote.NewRemote(&corev1.Secret{}, &corev1.SecretList{}, c)),
		TableConvertor: tables.CredentialConverter,
	}
}

func (s *Strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	s.PrepareForCreate(ctx, obj)
}

func (s *Strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	cred := obj.(*apiv1.Credential)
	cred.ServerAddress = normalizeDockerIO(cred.ServerAddress)
}
func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.Credential)
	if !params.NoValidate {
		if err := s.credentialValidate(ctx, params.Username, *params.Password, params.ServerAddress); err != nil {
			result = append(result, field.Invalid(field.NewPath("namespace"), params.Namespace, err.Error()))
		}
	}
	return result
}
func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.Credential)
	return s.Validate(ctx, params)
}
