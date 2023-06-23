package encryption

import (
	"context"
	"errors"

	apiv1 "github.com/acorn-io/runtime/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/runtime/pkg/encryption/nacl"
	"github.com/sirupsen/logrus"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetEncryptionKeyList(ctx context.Context, c kclient.Reader, namespace string) ([]apiv1.EncryptionKey, error) {
	var keyNotFound *nacl.ErrKeyNotFound

	keys, err := naclKeysToEncryptionKeyList(ctx, c, namespace)
	if errors.As(err, &keyNotFound) {
		logrus.Error("failed to get encryption keys", err)
		return []apiv1.EncryptionKey{}, nil
	}
	return keys, err
}

func naclKeysToEncryptionKeyList(ctx context.Context, c kclient.Reader, namespace string) ([]apiv1.EncryptionKey, error) {
	out := []apiv1.EncryptionKey{}
	values, err := nacl.GetAllNaclKeys(ctx, c, namespace)
	if err != nil {
		return out, err
	}
	for pubKey := range values {
		if pubKey == "primary" {
			continue
		}
		out = append(out, apiv1.EncryptionKey{
			KeyID:       pubKey,
			Annotations: map[string]string{},
		})
	}
	return out, nil
}
