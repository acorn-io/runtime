package nacl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"golang.org/x/crypto/nacl/box"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func DecryptNamespacedData(ctx context.Context, c kclient.Client, data []byte, namespace string) ([]byte, error) {
	keys, err := GetAllNaclKeys(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		data, err := key.Decrypt(data)
		if err == nil {
			return data, nil
		}
	}

	return nil, &ErrUnableToDecrypt{}
}

func (k *NaclKey) Decrypt(encData []byte) ([]byte, error) {
	pubkey, err := keyToBytes(k.PublicKey)
	if err != nil {
		return nil, err
	}

	privkey, err := keyToBytes(k.privateKey)
	if err != nil {
		return nil, err
	}

	preppedData, err := unwrapForDecryption(encData)
	if err != nil {
		return nil, err
	}

	encryptedData, ok := preppedData[k.PublicKey]
	if !ok {
		return nil, &ErrDecryptionKeyNotAvailable{}
	}

	decryptedBytes, ok := box.OpenAnonymous(nil, encryptedData, pubkey, privkey)
	if !ok {
		return nil, &ErrUnableToDecrypt{}
	}

	return decryptedBytes, nil
}

func unwrapForDecryption(data []byte) (map[string][]byte, error) {
	trimmedData := strings.TrimPrefix(string(data), "ACORNENC:")

	data, err := base64.StdEncoding.DecodeString(trimmedData)
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
		returnData[k], err = base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, err
		}
	}

	return returnData, err
}
