package info

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Get(ctx context.Context, reader kclient.Reader) (*apiv1.Info, error) {
	v := version.Get()
	controller := &appsv1.Deployment{}
	if err := reader.Get(ctx, router.Key(system.Namespace, system.ControllerName), controller); err != nil {
		return nil, err
	}

	apiServer := &appsv1.Deployment{}
	if err := reader.Get(ctx, router.Key(system.Namespace, system.APIServerName), apiServer); err != nil {
		return nil, err
	}

	raw, err := config.Incomplete(ctx, reader)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Get(ctx, reader)
	if err != nil {
		return nil, err
	}

	return &apiv1.Info{
		Spec: apiv1.InfoSpec{
			Version:         v.String(),
			Tag:             v.Tag,
			GitCommit:       v.Commit,
			Dirty:           v.Dirty,
			ControllerImage: controller.Spec.Template.Spec.Containers[0].Image,
			APIServerImage:  apiServer.Spec.Template.Spec.Containers[0].Image,
			Config:          *cfg,
			UserConfig:      *raw,
		},
	}, nil
}
