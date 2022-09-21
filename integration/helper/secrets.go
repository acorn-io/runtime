package helper

import (
	"testing"

	"github.com/acorn-io/acorn/pkg/client"
	"github.com/acorn-io/acorn/pkg/encryption/nacl"
)

func EncryptData(t *testing.T, client client.Client, keys []string, data string) string {
	t.Helper()
	var pubKeys []string

	if keys == nil {
		info, err := client.Info(GetCTX(t))
		if err != nil {
			t.Fatal(err)
			return ""
		}
		pubKeys = append(pubKeys, info.Spec.PublicKey)
	} else {
		pubKeys = keys
	}

	encData, err := nacl.MultipleKeyEncrypt(data, pubKeys)
	if err != nil {
		t.Fatal(err)
		return ""
	}
	output, err := encData.Marshal()
	if err != nil {
		t.Fatal(err)
		return ""
	}

	return output
}

func GetEncryptionKeys(t *testing.T, clients []client.Client) []string {
	t.Helper()
	var pubKeys []string

	for _, client := range clients {
		info, err := client.Info(GetCTX(t))
		if err != nil {
			t.Fatal(err)
			return nil
		}
		pubKeys = append(pubKeys, info.Spec.PublicKey)
	}

	return pubKeys
}
