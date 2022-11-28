package apps

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/autoupgrade"
	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/tables"
	"github.com/acorn-io/mink/pkg/db"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/strategy/remote"
	"github.com/acorn-io/mink/pkg/strategy/translation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	strategy.CompleteStrategy
	rest.TableConvertor

	client        kclient.Client
	clientFactory *client.Factory
}

func NewStrategy(c kclient.WithWatch, clientFactory *client.Factory, db *db.Factory) (strategy.CompleteStrategy, error) {
	storageStrategy, err := newStorageStrategy(c, db)
	if err != nil {
		return nil, err
	}
	return &Strategy{
		TableConvertor:   tables.AppConverter,
		CompleteStrategy: storageStrategy,
		client:           c,
		clientFactory:    clientFactory,
	}, nil
}

func newStorageStrategy(kclient kclient.WithWatch, db *db.Factory) (strategy.CompleteStrategy, error) {
	if db != nil {
		return db.NewDBStrategy(&apiv1.App{})
	}
	return translation.NewTranslationStrategy(
		&Translator{},
		remote.NewRemote(&v1.AppInstance{}, &v1.AppInstanceList{}, kclient)), nil
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	params := obj.(*apiv1.App)

	if err := s.createNamespace(ctx, params.Namespace); err != nil {
		result = append(result, field.Invalid(field.NewPath("namespace"), params.Namespace, err.Error()))
	}

	if _, isPattern := autoupgrade.AutoUpgradePattern(params.Spec.Image); !isPattern {
		image, local, err := s.resolveLocalImage(ctx, params.Namespace, params.Spec.Image)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
			return
		}

		if !local {
			if err := s.checkRemoteAccess(ctx, params.Namespace, image); err != nil {
				result = append(result, field.Invalid(field.NewPath("spec", "image"), params.Spec.Image, err.Error()))
				return
			}
		}

		permsFromImage, err := s.getPermissions(ctx, params.Namespace, image)
		if err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
			return
		}

		if err := s.checkRequestedPermsSatisfyImagePerms(permsFromImage, params.Spec.Permissions); err != nil {
			result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
			return
		}
	}

	if err := s.checkPermissionsForPrivilegeEscalation(ctx, params.Spec.Permissions); err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "permissions"), params.Spec.Permissions, err.Error()))
	}

	return result
}

func (s *Strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) (result field.ErrorList) {
	newParams := obj.(*apiv1.App)
	return s.Validate(ctx, newParams)
}

func (s *Strategy) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return tables.AppConverter.ConvertToTable(ctx, object, tableOptions)
}
