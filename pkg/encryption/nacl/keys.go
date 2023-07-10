package nacl

import (
	"context"
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/acorn-io/z"
	"golang.org/x/crypto/nacl/box"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	naclStoreKey = "nacl-keys"
	naclNSUID    = "ns-uid"
)

type NaclKeys map[string]*NaclKey
type NaclKey struct {
	AcornNamespace    string
	Primary           *bool
	PublicKey         *[32]byte
	acornNamespaceUID string
	privateKey        *[32]byte
}

type naclKeyStore map[string]naclStoredKey
type naclStoredKey struct {
	AcornNamespace    string    `json:"acornNamespace,omitempty"`
	Primary           *bool     `json:"primary,omitempty"`
	AcornNamespaceUID string    `json:"acornNamespaceUID,omitempty"`
	PrivateKey        *[32]byte `json:"privateKey,omitempty"`
	PublicKey         *[32]byte `json:"publicKey,omitempty"`
}

func GetOrCreatePrimaryNaclKey(ctx context.Context, c kclient.Client, namespace string) (*NaclKey, error) {
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

	return keys["primary"], nil
}

func GetPrimaryNaclKey(ctx context.Context, c kclient.Reader, publicKey, namespace string) (*NaclKey, error) {
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
		if key, ok := keys[publicKey]; ok {
			return key, nil
		}
		return nil, &ErrKeyNotFound{}
	}

	return keys["primary"], nil
}

func GetAllNaclKeys(ctx context.Context, c kclient.Reader, namespace string) (NaclKeys, error) {
	existing, err := getExistingSecret(ctx, c, namespace)
	if apierrors.IsNotFound(err) {
		return nil, &ErrKeyNotFound{}
	}
	if err != nil {
		return nil, err
	}

	return secretToNaclKeys(existing, namespace)
}

func GetPublicKey(ctx context.Context, c kclient.Reader, namespace string) (string, error) {
	key, err := GetPrimaryNaclKey(ctx, c, "", namespace)
	if key != nil {
		return KeyBytesToB64String(key.PublicKey), err
	}
	return "", err
}

func generateNewKeys(ctx context.Context, c kclient.Client, namespace string, existing *corev1.Secret) (*NaclKey, error) {
	naclKey := &NaclKey{
		AcornNamespace: namespace,
	}
	publicKey, privateKey, err := box.GenerateKey(crypto_rand.Reader)
	if err != nil {
		return nil, err
	}

	naclKey.PublicKey = publicKey
	naclKey.privateKey = privateKey

	ns := &corev1.Namespace{}
	err = c.Get(ctx, kclient.ObjectKey{
		Namespace: "",
		Name:      namespace,
	}, ns)
	if err != nil {
		return nil, err
	}

	naclKey.acornNamespaceUID = string(ns.UID)
	if existing == nil {
		naclKey.Primary = z.P(true)
	}

	return naclKey, createOrUpdateNaclKeySecret(ctx, c, naclKey, existing)
}

func getExistingSecret(ctx context.Context, c kclient.Reader, namespace string) (*corev1.Secret, error) {
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
				Name:      secretNameGen(key.AcornNamespace, key.acornNamespaceUID),
				Labels: map[string]string{
					labels.AcornSecretGenerated: "true",
					labels.AcornManaged:         "true",
				},
			},
			Type: corev1.SecretTypeOpaque,
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
	if bytes, err := base64.RawURLEncoding.DecodeString(key); err != nil {
		return nil, err
	} else {
		copy(returnBytes[:], bytes)
	}
	return returnBytes, nil
}

func KeyBytesToB64String(key *[32]byte) string {
	bytes := key[:]
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func secretToNaclKeys(secret *corev1.Secret, namespace string) (NaclKeys, error) {
	to := NaclKeys{}

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

	for pubKeyString, keyInfo := range store {
		pubKey, err := keyToBytes(pubKeyString)
		if err != nil {
			return nil, err
		}
		to[pubKeyString] = &NaclKey{
			AcornNamespace:    keyInfo.AcornNamespace,
			Primary:           keyInfo.Primary,
			PublicKey:         pubKey,
			privateKey:        keyInfo.PrivateKey,
			acornNamespaceUID: string(uid),
		}
		if *keyInfo.Primary {
			to["primary"] = to[pubKeyString]
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

	if existingData != nil {
		if keydata, ok := existingData[naclStoreKey]; ok {
			err = json.Unmarshal(keydata, &store)
			if err != nil {
				return nil, err
			}
		}
	}
	stringKey := KeyBytesToB64String(k.PublicKey)
	store[stringKey] = naclStoredKey{
		AcornNamespace:    k.AcornNamespace,
		Primary:           k.Primary,
		AcornNamespaceUID: k.acornNamespaceUID,
		PrivateKey:        k.privateKey,
		PublicKey:         k.PublicKey,
	}

	to[naclStoreKey], err = json.Marshal(store)
	return to, err
}

func naclk8sKey(namespace, name string) kclient.ObjectKey {
	return kclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
}

func naclSecretName(ctx context.Context, c kclient.Reader, namespace string) (string, error) {
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
