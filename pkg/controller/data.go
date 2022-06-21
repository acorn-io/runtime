package controller

import (
	"context"

	"github.com/acorn-io/acorn/pkg/build/buildkit"
	"github.com/acorn-io/acorn/pkg/config"
	"github.com/acorn-io/acorn/pkg/system"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) initData(ctx context.Context) error {
	err := c.apply.WithSetID("acorn-controller-data").ApplyObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: system.Namespace,
			},
		})
	if err != nil {
		return err
	}
	if err := config.Init(ctx, c.client); err != nil {
		return err
	}
	return buildkit.SyncBuildkitPod(ctx, c.client)
}
