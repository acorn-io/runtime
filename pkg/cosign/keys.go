package cosign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

var (
	supportedKeyTypes = map[string]interface{}{
		ssh.KeyAlgoRSA:      nil,
		ssh.KeyAlgoED25519:  nil,
		ssh.KeyAlgoECDSA256: nil,
		ssh.KeyAlgoECDSA384: nil,
		ssh.KeyAlgoECDSA521: nil,
	}
)

func PemEncodeCryptoPublicKey(pubKey crypto.PublicKey) ([]byte, string, error) {
	var encoded []byte
	var b []byte
	var err error
	switch pubKey := pubKey.(type) {
	case *rsa.PublicKey:
		b = x509.MarshalPKCS1PublicKey(pubKey)
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: b,
		})
	case ed25519.PublicKey:
		b, err = x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return nil, "", err
		}
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PUBLIC KEY",
			Bytes: b,
		})
	case *ecdsa.PublicKey:
		b, err = x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return nil, "", err
		}
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "ECDSA PUBLIC KEY",
			Bytes: b,
		})
	default:
		return nil, "", fmt.Errorf("unsupported key type %T", pubKey)
	}

	hash := sha256.Sum256(encoded)
	fingerprint := hex.EncodeToString(hash[:])

	return encoded, fingerprint, nil
}

func PemEncodeSSHPublicKey(key ssh.PublicKey) ([]byte, error) {
	pubKey := key.(ssh.CryptoPublicKey).CryptoPublicKey()
	pem, _, err := PemEncodeCryptoPublicKey(pubKey)
	return pem, err
}

func ParsePublicKey(keystr string) (crypto.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keystr)
	if err != nil {
		return nil, err
	}

	parsedKey, err := ssh.ParsePublicKey(keyBytes)
	if err != nil {
		return nil, err
	}

	if _, ok := supportedKeyTypes[parsedKey.Type()]; !ok {
		return nil, fmt.Errorf("Unsupported key type '%s'", parsedKey.Type())
	}

	return parsedKey.(ssh.CryptoPublicKey).CryptoPublicKey(), nil
}
