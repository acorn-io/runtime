package pullsecret

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/authn/kubernetes"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	SharedPullSecret = "shared-pull-secret"
)

func ForNamespace(ctx context.Context, c client.Reader, namespace string, secretNames ...string) ([]corev1.Secret, error) {
	var secrets []corev1.Secret
	for _, name := range append(secretNames, SharedPullSecret) {
		secret := corev1.Secret{}
		err := c.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: namespace,
		}, &secret)
		if apierrors.IsNotFound(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		if secret.Type == corev1.SecretTypeDockerConfigJson {
			secrets = append(secrets, secret)
		}
	}
	return secrets, nil
}

func Keychain(ctx context.Context, c client.Reader, namespace string, secretNames ...string) (authn.Keychain, error) {
	secrets, err := ForNamespace(ctx, c, namespace, secretNames...)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, secret := range secrets {
		names = append(names, secret.Name)
	}

	auth, err := k8schain.NewInCluster(ctx, k8schain.Options{
		Namespace:        namespace,
		ImagePullSecrets: names,
	})
	if err == nil {
		return auth, nil
	}
	return kubernetes.NewFromPullSecrets(ctx, secrets)
}
