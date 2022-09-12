package nacl

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

type UnableToDecrypt struct{}
type DecryptionKeyNotAvailable struct{}

func (utd *UnableToDecrypt) Error() string {
	return "Unable to decrypt values"
}

func (d *DecryptionKeyNotAvailable) Error() string {
	return "Decryption Key Not Available on this Cluster"
}

func (k *ClusterKey) Decrypt(encData []byte) ([]byte, error) {
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
		return nil, &DecryptionKeyNotAvailable{}
	}

	decryptedBytes, ok := box.OpenAnonymous(nil, encryptedData, pubkey, privkey)
	if !ok {
		return nil, &UnableToDecrypt{}
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
