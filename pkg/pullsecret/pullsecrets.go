package pullsecret

import (
	"context"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"github.com/ibuildthecloud/baaah/pkg/meta"
	"github.com/ibuildthecloud/baaah/pkg/router"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	SharedPullSecret = "shared-pull-secret"
)

func ForNamespace(c router.Getter, namespace string, secretNames ...string) ([]corev1.Secret, error) {
	var secrets []corev1.Secret
	for _, name := range append(secretNames, SharedPullSecret) {
		secret := corev1.Secret{}
		err := c.Get(&secret, name, &meta.GetOptions{
			Namespace: namespace,
		})
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

func Keychain(ctx context.Context, c router.Getter, namespace string, secretNames ...string) (authn.Keychain, error) {
	secrets, err := ForNamespace(c, namespace, secretNames...)
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
