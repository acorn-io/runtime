package pullsecret

import (
	"context"
	"encoding/json"
	"fmt"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/dockerconfig"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/kubernetes"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ForNamespace(ctx context.Context, c client.Reader, namespace string, requireLabel bool) ([]corev1.Secret, error) {
	secrets := &corev1.SecretList{}
	err := c.List(ctx, secrets, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	var result []corev1.Secret
	for _, secret := range secrets.Items {
		if requireLabel && secret.Labels[labels.AcornCredential] != "true" {
			continue
		}
		if secret.Type == corev1.SecretTypeDockercfg || secret.Type == corev1.SecretTypeDockerConfigJson {
			result = append(result, secret)
			continue
		} else if secret.Type == apiv1.SecretTypeCredential {
			data, err := nacl.DecryptNamespacedDataMap(ctx, c, secret.Data, secret.Namespace)
			if err != nil {
				return nil, fmt.Errorf("decrypting acorn credential %s/%s: %w", secret.Namespace, secret.Name, err)
			}
			secret, err := dockerconfig.FromCredentialData(data)
			if err != nil {
				return nil, err
			}
			result = append(result, *secret)
		}
	}

	for i, secret := range result {
		result[i].Data, err = nacl.DecryptNamespacedDataMap(ctx, c, secret.Data, secret.Namespace)
		if err != nil {
			return nil, fmt.Errorf("final decrypting %s/%s: %w", secret.Namespace, secret.Name, err)
		}
	}

	return result, nil
}

func Keychain(ctx context.Context, c client.Reader, namespace string) (authn.Keychain, error) {
	keychainSecrets, err := ForNamespace(ctx, c, system.ImagesNamespace, true)
	if err != nil {
		return nil, err
	}

	if namespace != "" {
		secrets, err := ForNamespace(ctx, c, namespace, false)
		if err != nil {
			return nil, err
		}
		keychainSecrets = append(keychainSecrets, secrets...)
	}
	return kubernetes.NewFromPullSecrets(ctx, keychainSecrets)
}

func ForImages(secretName, secretNamespace string, keychain authn.Keychain, images ...string) (*corev1.Secret, error) {
	dockerConfig := map[string]any{}

	for _, image := range images {
		ref, err := name.ParseReference(image)
		if err != nil {
			return nil, err
		}

		registry := ref.Context().RegistryStr()

		auth, err := keychain.Resolve(ref.Context())
		if err != nil {
			return nil, err
		}

		config, err := auth.Authorization()
		if err != nil {
			return nil, err
		}

		dockerConfig[registry] = config
	}

	data, err := json.Marshal(map[string]any{
		"auths": dockerConfig,
	})
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			Labels: map[string]string{
				labels.AcornManaged:    "true",
				labels.AcornPullSecret: "true",
			},
		},
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: data,
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}, nil
}
