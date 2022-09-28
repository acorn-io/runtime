package encryption

import (
	"context"
	"errors"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
	"github.com/sirupsen/logrus"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetEncryptionKeySpecList(ctx context.Context, c kclient.Reader, namespace string) ([]apiv1.EncryptionKeySpec, error) {
	var keyNotFound *nacl.ErrKeyNotFound

	keys, err := naclKeysToEncryptionKeySpecList(ctx, c, namespace)
	if errors.As(err, &keyNotFound) {
		logrus.Error(err)
		return []apiv1.EncryptionKeySpec{}, nil
	}
	return keys, err
}

func naclKeysToEncryptionKeySpecList(ctx context.Context, c kclient.Reader, namespace string) ([]apiv1.EncryptionKeySpec, error) {
	out := []apiv1.EncryptionKeySpec{}
	values, err := nacl.GetAllNaclKeys(ctx, c, namespace)
	if err != nil {
		return out, err
	}
	for pubKey := range values {
		if pubKey == "primary" {
			continue
		}
		out = append(out, apiv1.EncryptionKeySpec{
			KeyID:       pubKey,
			Annotations: map[string]string{},
		})
	}
	return out, nil
}
