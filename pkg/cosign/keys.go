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
	"regexp"

	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"golang.org/x/crypto/ssh"
)

var (
	supportedSSHKeyAlgos = map[string]struct{}{
		ssh.KeyAlgoRSA:      {},
		ssh.KeyAlgoED25519:  {},
		ssh.KeyAlgoECDSA256: {},
		ssh.KeyAlgoECDSA384: {},
		ssh.KeyAlgoECDSA521: {},
	}

	PubkeyPrefixPattern = regexp.MustCompile(`^-----BEGIN (RSA |ED25519 |ECDSA ){0,1}PUBLIC KEY-----\n(.*\n)+-----END (RSA |ED25519 |ECDSA ){0,1}PUBLIC KEY-----\s*$`)
)

func PemEncodeCryptoPublicKey(pubKey crypto.PublicKey) ([]byte, string, error) {
	encoded, err := cryptoutils.MarshalPublicKeyToPEM(pubKey)
	if err != nil {
		return nil, "", err
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

func ParseSSHPublicKey(keystr string) (crypto.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(keystr)
	if err != nil {
		return nil, err
	}

	parsedKey, err := ssh.ParsePublicKey(keyBytes)
	if err != nil {
		return nil, err
	}

	if _, ok := supportedSSHKeyAlgos[parsedKey.Type()]; !ok {
		return nil, fmt.Errorf("Unsupported key type '%s'", parsedKey.Type())
	}

	return parsedKey.(ssh.CryptoPublicKey).CryptoPublicKey(), nil
}

// UnmarshalPEMToPublicKey converts a PEM-encoded byte slice into a crypto.PublicKey
func UnmarshalPEMToPublicKey(pemBytes []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("PEM decoding failed")
	}

	switch block.Type {
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "ECDSA PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if ecdsaPub, ok := pub.(*ecdsa.PublicKey); ok {
			return ecdsaPub, nil
		}
	case "PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		switch key := pub.(type) {
		case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey:
			return key, nil
		}
	case "ED25519 PUBLIC KEY":
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		if ed25519Pub, ok := pub.(ed25519.PublicKey); ok {
			return ed25519Pub, nil
		}
	}

	return nil, fmt.Errorf("unsupported public key type or format")
}
