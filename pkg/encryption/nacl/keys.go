package nacl

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

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
	naclStoreKey = "enc-keys"
	naclNSUID    = "ns-uid"
)

type naclKeyStore []map[string][]byte

type NaclKey struct {
	PublicKey         string
	acornNamespace    string
	acornNamespaceUID string
	privateKey        string
}

func GetOrCreateNaclKey(ctx context.Context, c kclient.Client, namespace string) (*NaclKey, error) {
	existing, err := getExistingSecret(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return generateNewKeys(ctx, c, namespace, nil)
	} else if !apierrors.IsNotFound(err) && err != nil {
		return nil, err
	}

	keys, err := secretToNaclKeys(existing, namespace)
	if err != nil {
		return nil, err
	}

	return keys[0], nil
}

func GetNaclKey(ctx context.Context, c kclient.Client, publicKey, namespace string) (*NaclKey, error) {
	existing, err := getExistingSecret(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return nil, &ErrKeyNotFound{}
	} else if err != nil {
		return nil, err
	}

	keys, err := secretToNaclKeys(existing, namespace)
	if err != nil {
		return nil, err
	}

	if publicKey != "" {
		for _, key := range keys {
			if key.PublicKey == publicKey {
				return key, nil
			}
		}
	}

	return keys[0], nil
}

func GetAllNaclKey(ctx context.Context, c kclient.Client, namespace string) ([]*NaclKey, error) {
	existing, err := getExistingSecret(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return nil, &ErrKeyNotFound{}
	}
	if err != nil {
		return nil, err
	}

	return secretToNaclKeys(existing, namespace)
}

func GetPublicKey(ctx context.Context, c kclient.Client, namespace string) (string, error) {
	key, err := GetOrCreateNaclKey(ctx, c, namespace)
	if key != nil {
		return key.PublicKey, err
	}
	return "", err
}

func generateNewKeys(ctx context.Context, c kclient.Client, namespace string, existing *corev1.Secret) (*NaclKey, error) {
	naclKey := &NaclKey{
		acornNamespace: namespace,
	}
	publicKey, privateKey, err := box.GenerateKey(crypto_rand.Reader)
	if err != nil {
		return nil, err
	}

	naclKey.PublicKey = base64.StdEncoding.EncodeToString(publicKey[:])
	naclKey.privateKey = base64.StdEncoding.EncodeToString(privateKey[:])

	ns := &corev1.Namespace{}
	err = c.Get(ctx, kclient.ObjectKey{
		Namespace: "",
		Name:      namespace,
	}, ns)
	if err != nil {
		return nil, err
	}

	naclKey.acornNamespaceUID = string(ns.UID)

	return naclKey, createOrUpdateNaclKeySecret(ctx, c, naclKey, existing)
}

func getExistingSecret(ctx context.Context, c kclient.Client, namespace string) (*corev1.Secret, error) {
	nsString, err := naclSecretName(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	existing := &corev1.Secret{}
	err = c.Get(ctx, naclk8sKey(system.Namespace, nsString), existing)
	return existing, err
}

func createOrUpdateNaclKeySecret(ctx context.Context, c kclient.Client, key *NaclKey, existing *corev1.Secret) error {
	if existing == nil {
		keyData, err := key.toSecretData(nil)
		if err != nil {
			return err
		}

		return c.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace,
				Name:      secretNameGen(key.acornNamespace, key.acornNamespaceUID),
				Labels: map[string]string{
					labels.AcornSecretGenerated: "true",
					labels.AcornManaged:         "true",
				},
			},
			Type: v1.SecretTypeOpaque,
			Data: keyData,
		})
	}

	keyData, err := key.toSecretData(existing.Data)
	if err != nil {
		return err
	}

	updatedSecret := existing.DeepCopy()
	updatedSecret.Data = keyData

	return c.Update(ctx, updatedSecret)
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

func secretToNaclKeys(secret *corev1.Secret, namespace string) ([]*NaclKey, error) {
	to := []*NaclKey{}

	keystore, ok := secret.Data[naclStoreKey]
	if !ok {
		return nil, NewErrKeyNotFound(true)
	}
	uid, ok := secret.Data[naclNSUID]
	if !ok {
		return nil, fmt.Errorf("UID for Acorn namespace %s not found", namespace)
	}

	store := naclKeyStore{}

	err := json.Unmarshal(keystore, &store)
	if err != nil {
		return nil, err
	}

	if len(store) == 0 {
		return nil, NewErrKeyNotFound(false)
	}

	for _, keyInfo := range store {
		for pub, priv := range keyInfo {
			to = append(to, &NaclKey{
				acornNamespace:    namespace,
				PublicKey:         pub,
				privateKey:        string(priv),
				acornNamespaceUID: string(uid),
			})
		}
	}

	return to, nil
}

func (k *NaclKey) toSecretData(existingData map[string][]byte) (map[string][]byte, error) {
	to := map[string][]byte{
		naclNSUID: []byte(k.acornNamespaceUID),
	}
	store := naclKeyStore{}
	var err error

	if existingData == nil {
		store = append(store, map[string][]byte{k.PublicKey: []byte(k.privateKey)})
		to[naclStoreKey], err = json.Marshal(store)
		return to, err
	}

	if keydata, ok := existingData[naclStoreKey]; ok {
		err = json.Unmarshal(keydata, &store)
		if err != nil {
			return nil, err
		}
	}

	store = append(store, map[string][]byte{k.PublicKey: []byte(k.privateKey)})
	to[naclStoreKey], err = json.Marshal(store)
	return to, err
}

func naclk8sKey(namespace, name string) kclient.ObjectKey {
	return kclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
}

func naclSecretName(ctx context.Context, c kclient.Client, namespace string) (string, error) {
	ns := &corev1.Namespace{}
	err := c.Get(ctx, kclient.ObjectKey{
		Name: namespace,
	}, ns)
	if err != nil {
		return "", err
	}
	return secretNameGen(namespace, string(ns.UID)), nil
}

func secretNameGen(namespace, uid string) string {
	if len(uid) > 11 {
		return fmt.Sprintf("%s-%s-enc-keys", namespace, uid[:12])
	}
	return fmt.Sprintf("%s-%s-enc-keys", namespace, uid)
}
