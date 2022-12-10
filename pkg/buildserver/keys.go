package buildserver

import (
	"encoding/base64"
	"fmt"
)

func ToKey(key string) (result [32]byte, err error) {
	slice, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return
	}
	if len(slice) != 32 {
		return result, fmt.Errorf("invalid key length [%d] expected 32 bytes", len(slice))
	}
	copy(result[:], slice)
	return
}

func ToKeys(pub, priv string) (pubKey [32]byte, privKey [32]byte, err error) {
	pubKey, err = ToKey(pub)
	if err != nil {
		return pubKey, privKey, err
	}

	privKey, err = ToKey(priv)
	return pubKey, privKey, err
}
