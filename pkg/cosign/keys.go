package cosign

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

func PemEncodeSSHPublicKey(key ssh.PublicKey) ([]byte, error) {
	pubKey := key.(ssh.CryptoPublicKey).CryptoPublicKey()
	var encoded []byte
	switch pubKey := pubKey.(type) {
	case *rsa.PublicKey:
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: x509.MarshalPKCS1PublicKey(pubKey),
		})
	case ed25519.PublicKey:
		b, err := x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return nil, err
		}
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "ED25519 PUBLIC KEY",
			Bytes: b,
		})
	case *ecdsa.PublicKey:
		b, err := x509.MarshalPKIXPublicKey(pubKey)
		if err != nil {
			return nil, err
		}
		encoded = pem.EncodeToMemory(&pem.Block{
			Type:  "ECDSA PUBLIC KEY",
			Bytes: b,
		})
	default:
		return nil, fmt.Errorf("unsupported key type %T", pubKey)
	}

	return encoded, nil
}
