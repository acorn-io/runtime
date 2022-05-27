package secrets

import (
	"context"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewExpose(c client.WithWatch) *Expose {
	return &Expose{
		client: c,
	}
}

type Expose struct {
	client client.WithWatch
}

func (s *Expose) NamespaceScoped() bool {
	return true
}

func (s *Expose) New() runtime.Object {
	return &apiv1.Secret{}
}

func (s *Expose) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	ns, _ := request.NamespaceFrom(ctx)

	secret := &corev1.Secret{}
	err := s.client.Get(ctx, router.Key(ns, name), secret)
	if err != nil {
		return nil, err
	}

	newSecret := coreSecretToSecret(secret.DeepCopy())
	newSecret.Data = secret.Data
	return newSecret, nil
}
