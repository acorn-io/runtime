package info

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/acorn/pkg/version"
	"github.com/acorn-io/baaah/pkg/router"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/endpoints/request"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Get(ctx context.Context, c kclient.Client) (*apiv1.Info, error) {
	var controllerImage string
	var apiServerImage string

	v := version.Get()

	controller := &appsv1.Deployment{}
	if err := c.Get(ctx, router.Key(system.Namespace, system.ControllerName), controller); !apierrors.IsNotFound(err) && err != nil {
		return nil, err
	} else if err == nil {
		controllerImage = controller.Spec.Template.Spec.Containers[0].Image
	}

	apiServer := &appsv1.Deployment{}
	if err := c.Get(ctx, router.Key(system.Namespace, system.APIServerName), apiServer); !apierrors.IsNotFound(err) && err != nil {
		return nil, err
	} else if err == nil {
		apiServerImage = apiServer.Spec.Template.Spec.Containers[0].Image
	}

	raw, err := config.Incomplete(ctx, c)
	if err != nil {
		return nil, err
	}

	cfg, err := config.Get(ctx, c)
	if err != nil {
		return nil, err
	}
	ns, _ := request.NamespaceFrom(ctx)

	pubKey, err := nacl.GetPublicKey(ctx, c, ns)
	if err != nil {
		return nil, err
	}

	return &apiv1.Info{
		Spec: apiv1.InfoSpec{
			PublicKey:       pubKey,
			Version:         v.String(),
			Tag:             v.Tag,
			GitCommit:       v.Commit,
			Dirty:           v.Dirty,
			ControllerImage: controllerImage,
			APIServerImage:  apiServerImage,
			Config:          *cfg,
			UserConfig:      *raw,
		},
	}, nil
}
