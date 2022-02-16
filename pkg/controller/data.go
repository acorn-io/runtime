package controller

import (
	"context"

	"github.com/ibuildthecloud/herd/pkg/system"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Controller) initData(ctx context.Context) error {
	return c.apply.WithSetID("herd-controller-data").ApplyObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: system.Namespace,
			},
		})
}
