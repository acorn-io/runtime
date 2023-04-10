package nacl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/nacl/box"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	EncPrefix = "ACORNENC:"
	EncSuffix = "::"
)

func IsAcornEncryptedData(data []byte) bool {
	return strings.HasPrefix(string(data), EncPrefix)
}

func DecryptNamespacedDataMap(ctx context.Context, c kclient.Reader, data map[string][]byte, ownerNamespace string) (map[string][]byte, error) {
	to := map[string][]byte{}
	for k, v := range data {
		if IsAcornEncryptedData(v) {
			decryptedData, err := DecryptNamespacedData(ctx, c, v, ownerNamespace)
			if err != nil {
				return data, err
			}
			to[k] = decryptedData

			continue
		}
		to[k] = v
	}

	return to, nil
}

func DecryptNamespacedData(ctx context.Context, c kclient.Reader, data []byte, namespace string) ([]byte, error) {
	keys, err := GetAllNaclKeys(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	var errs []error
	for _, key := range keys {
		data, err := key.Decrypt(data)
		if err == nil {
			return data, nil
		} else {
			errs = append(errs, fmt.Errorf("pubkey %s: %w", KeyBytesToB64String(key.PublicKey), err))
		}
	}

	return nil, &ErrUnableToDecrypt{
		Errs: errs,
	}
}

func (k *NaclKey) Decrypt(encData []byte) ([]byte, error) {
	pubKeyString := KeyBytesToB64String(k.PublicKey)
	preppedData, err := unwrapForDecryption(encData)
	if err != nil {
		return nil, err
	}

	encryptedData, ok := preppedData[pubKeyString]
	if !ok {
		return nil, &ErrDecryptionKeyNotAvailable{}
	}

	decryptedBytes, ok := box.OpenAnonymous(nil, encryptedData, k.PublicKey, k.privateKey)
	if !ok {
		return nil, &ErrUnableToDecrypt{}
	}

	return decryptedBytes, nil
}

func unwrapForDecryption(data []byte) (map[string][]byte, error) {
	trimmedData := strings.TrimPrefix(string(data), EncPrefix)
	trimmedData = strings.TrimSuffix(trimmedData, EncSuffix)

	data, err := base64.RawURLEncoding.DecodeString(trimmedData)
	if err != nil {
		return nil, err
	}

	mappedData := &map[string]string{}
	err = json.Unmarshal(data, mappedData)
	if err != nil {
		return nil, err
	}

	returnData := map[string][]byte{}
	for k, v := range *mappedData {
		returnData[k], err = base64.RawURLEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
	}

	return returnData, err
}
