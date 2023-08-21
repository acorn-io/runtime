package builder

import (
	"context"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/system"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func getDepotKey(ctx context.Context, c kclient.Client, namespace string) (string, string, error) {
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return "", "", err
	}

	if *cfg.InternalRegistryPrefix == "" {
		return "", "", nil
	}

	sec := &corev1.Secret{}
	if err := c.Get(ctx, router.Key(namespace, "depot-builder-key"), sec); apierrors.IsNotFound(err) {
		if err := c.Get(ctx, router.Key(system.ImagesNamespace, "depot-builder-key"), sec); apierrors.IsNotFound(err) {
			return "", "", nil
		}
		return "", "", nil
	} else if err != nil {
		return "", "", err
	}
	return string(sec.Data["token"]), string(sec.Data["projectId"]), nil
}
