package nacl

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"golang.org/x/crypto/nacl/box"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	acornClusterKeySecretName = "acorn-encryption-keys"
)

type ClusterKey struct {
	PublicKey  string
	privateKey string
}

type KeyNotFound struct{}

func (k *KeyNotFound) Error() string {
	return "No Key Found"
}

func GetOrCreateClusterKey(ctx context.Context, c kclient.Client) (*ClusterKey, error) {
	existingSecret := &corev1.Secret{}
	err := c.Get(ctx, k8sKey(system.Namespace, acornClusterKeySecretName), existingSecret)
	if apierrors.IsNotFound(err) {
		return generateNewKeys(ctx, c)
	}
	if !apierrors.IsNotFound(err) && err != nil {
		return nil, err
	}

	return secretToClusterKey(existingSecret)
}

func generateNewKeys(ctx context.Context, c kclient.Client) (*ClusterKey, error) {
	clusterKey := &ClusterKey{}
	publicKey, privateKey, err := box.GenerateKey(crypto_rand.Reader)
	if err != nil {
		return nil, err
	}

	clusterKey.PublicKey = base64.StdEncoding.EncodeToString(publicKey[:])
	clusterKey.privateKey = base64.StdEncoding.EncodeToString(privateKey[:])

	return clusterKey, writeClusterKeySecret(ctx, c, clusterKey)
}

func writeClusterKeySecret(ctx context.Context, c kclient.Client, masterKey *ClusterKey) error {
	keyData, err := masterKeyToSecretData(masterKey)
	if err != nil {
		return err
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: system.Namespace,
			Name:      acornClusterKeySecretName,
			Labels: map[string]string{
				labels.AcornSecretGenerated: "true",
				labels.AcornManaged:         "true",
			},
		},
		Type: v1.SecretTypeOpaque,
		Data: keyData,
	}

	return c.Create(ctx, secret)
}

func keyToBytes(key string) (*[32]byte, error) {
	returnBytes := &[32]byte{}
	if bytes, err := base64.StdEncoding.DecodeString(key); err != nil {
		return nil, err
	} else {
		copy(returnBytes[:], bytes)
	}
	return returnBytes, nil
}

func secretToClusterKey(secret *corev1.Secret) (*ClusterKey, error) {
	to := &ClusterKey{}

	publicKey, ok := secret.Data["public-key"]
	if !ok {
		return to, &KeyNotFound{}
	}
	to.PublicKey = string(publicKey)

	privateKey, ok := secret.Data["private-key"]
	if !ok {
		return to, &KeyNotFound{}
	}
	to.privateKey = string(privateKey)

	return to, nil
}

func masterKeyToSecretData(masterKey *ClusterKey) (map[string][]byte, error) {
	to := map[string][]byte{}

	to["public-key"] = []byte(masterKey.PublicKey)
	to["private-key"] = []byte(masterKey.privateKey)

	return to, nil
}

func GetPublicKey(ctx context.Context, c kclient.Reader) (string, error) {
	keySecret := &corev1.Secret{}
	err := c.Get(ctx, k8sKey(system.Namespace, acornClusterKeySecretName), keySecret)
	if err != nil {
		return "", err
	}

	masterKey, err := secretToClusterKey(keySecret)
	if err != nil {
		return "", err
	}

	return masterKey.PublicKey, nil
}

func k8sKey(namespace, name string) kclient.ObjectKey {
	return kclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
}
