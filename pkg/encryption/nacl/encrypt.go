package nacl

import (
	crypto_rand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"

	"golang.org/x/crypto/nacl/box"
)

type EncryptedData struct {
	PublicKey        string `json:"publicKey"`
	EncryptedContent string `json:"encryptedContent"`
}

type MultiEncryptedData map[string]string

func Encrypt(msg, recipientPublicKey string) (*EncryptedData, error) {
	key, err := keyToBytes(recipientPublicKey)
	if err != nil {
		return nil, err
	}

	encryptedBytes, err := box.SealAnonymous(nil, []byte(msg), key, crypto_rand.Reader)
	return &EncryptedData{
		PublicKey:        recipientPublicKey,
		EncryptedContent: base64.StdEncoding.EncodeToString(encryptedBytes),
	}, err
}

func MultipleKeyEncrypt(msg string, keys []string) (MultiEncryptedData, error) {
	outputData := MultiEncryptedData{}
	for _, pubKey := range keys {
		encData, err := Encrypt(msg, pubKey)
		if err != nil {
			return outputData, err
		}
		outputData[pubKey] = encData.EncryptedContent
	}
	return outputData, nil
}

func (f *EncryptedData) Marshal() (string, error) {
	data := map[string]string{
		f.PublicKey: f.EncryptedContent,
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(jsonBytes)
	return strings.Join([]string{"ACORNENC", b64}, ":"), nil
}

func (f MultiEncryptedData) Marshal() (string, error) {
	jsonBytes, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	b64 := base64.StdEncoding.EncodeToString(jsonBytes)
	return strings.Join([]string{"ACORNENC", b64}, ":"), nil
}
