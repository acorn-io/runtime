package builds

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/mink/pkg/strategy"
	"github.com/acorn-io/mink/pkg/types"
	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/buildserver"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/imagesystem"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Strategy struct {
	client  kclient.Client
	creator strategy.Creater
}

func (s *Strategy) Validate(ctx context.Context, obj runtime.Object) (result field.ErrorList) {
	acornBuild := obj.(*apiv1.AcornImageBuild)
	builder := &apiv1.Builder{}

	err := s.client.Get(ctx, router.Key(acornBuild.Namespace, acornBuild.Spec.BuilderName), builder)
	if err != nil {
		result = append(result, field.Invalid(field.NewPath("spec", "builderName"), acornBuild.Spec.BuilderName, err.Error()))
		return
	}

	if builder.Status.PublicKey == "" || !builder.Status.Ready {
		result = append(result, field.Invalid(field.NewPath("spec", "builderName"), acornBuild.Spec.BuilderName, "builder is not ready"))
	}

	return
}

func (s *Strategy) Create(ctx context.Context, object types.Object) (types.Object, error) {
	acornBuild := object.(*apiv1.AcornImageBuild)
	builder := &apiv1.Builder{}

	err := s.client.Get(ctx, router.Key(acornBuild.Namespace, acornBuild.Spec.BuilderName), builder)
	if err != nil {
		return nil, err
	}

	pushRepo, err := imagesystem.GetBuildPushRepoForNamespace(ctx, s.client, acornBuild.Namespace)
	if err != nil {
		return nil, err
	}

	token, err := buildserver.CreateToken(builder, acornBuild, pushRepo.String())
	if err != nil {
		return nil, err
	}

	cfg, err := config.Get(ctx, s.client)
	if err != nil {
		return nil, err
	}

	if *cfg.RecordBuilds {
		result, err := s.creator.Create(ctx, object)
		if err != nil {
			return nil, err
		}
		acornBuild = result.(*apiv1.AcornImageBuild)
	}

	acornBuild.Status.BuildURL = builder.Status.Endpoint
	acornBuild.Status.Token = token
	return acornBuild, nil
}

func (s *Strategy) New() types.Object {
	return s.creator.New()
}
