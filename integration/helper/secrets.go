package helper

import (
	"testing"

	"github.com/acorn-io/runtime/pkg/client"
	"github.com/acorn-io/runtime/pkg/encryption/nacl"
)

func EncryptData(t *testing.T, client client.Client, keys []string, data string) string {
	t.Helper()
	var pubKeys []string

	if keys == nil {
		fullInfo, err := client.Info(GetCTX(t))
		if err != nil {
			t.Fatal(err)
			return ""
		}
		for _, info := range fullInfo {
			for _, region := range info.Regions {
				for _, key := range region.PublicKeys {
					pubKeys = append(pubKeys, key.KeyID)
				}
			}
		}
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
		fullInfo, err := client.Info(GetCTX(t))
		if err != nil {
			t.Fatal(err)
			return nil
		}
		for _, info := range fullInfo {
			for _, region := range info.Regions {
				for _, key := range region.PublicKeys {
					pubKeys = append(pubKeys, key.KeyID)
				}
			}
		}
	}

	return pubKeys
}
