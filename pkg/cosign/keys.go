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
	"os"
	"path/filepath"
	"regexp"

	"github.com/secure-systems-lab/go-securesystemslib/encrypted"
	"github.com/sigstore/cosign/v2/pkg/cosign"
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

/*
  Adapted from sigstore/cosign to include support for OPENSSH PRIVATE KEY
	Source: https://github.com/sigstore/cosign/blob/b43ce66500a808b932392557fb95f668625c4dbb/pkg/cosign/keys.go#L78-L181
*/

type Keys struct {
	private crypto.PrivateKey
	public  crypto.PublicKey
}

type KeysBytes struct {
	PrivateBytes []byte
	PublicBytes  []byte
	password     []byte
}

func (k *KeysBytes) Password() []byte {
	return k.password
}

func ImportKeyPair(keyPath string, pass []byte) (*KeysBytes, error) {
	pemBytes, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return nil, err
	}

	pemBlock, _ := pem.Decode(pemBytes)
	if pemBlock == nil {
		return nil, fmt.Errorf("invalid pem block")
	}

	var signer crypto.Signer

	switch pemBlock.Type {
	case cosign.RSAPrivateKeyPemType:
		rsaPk, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing rsa private key: %w", err)
		}
		if err = cryptoutils.ValidatePubKey(rsaPk.Public()); err != nil {
			return nil, fmt.Errorf("error validating rsa key: %w", err)
		}
		signer = rsaPk
	case cosign.ECPrivateKeyPemType:
		ecdsaPk, err := x509.ParseECPrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing ecdsa private key")
		}
		if err = cryptoutils.ValidatePubKey(ecdsaPk.Public()); err != nil {
			return nil, fmt.Errorf("error validating ecdsa key: %w", err)
		}
		signer = ecdsaPk
	case cosign.PrivateKeyPemType:
		pkcs8Pk, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("error parsing pkcs #8 private key")
		}
		signer, err = getSigner(pkcs8Pk)
		if err != nil {
			return nil, err
		}
	case "OPENSSH PRIVATE KEY":
		var (
			err error
			key crypto.PrivateKey
		)
		if pass != nil {
			key, err = ssh.ParseRawPrivateKeyWithPassphrase(pemBytes, pass)
		} else {
			key, err = ssh.ParseRawPrivateKey(pemBytes)
		}
		if err != nil {
			return nil, fmt.Errorf("error parsing private key: %w", err)
		}
		signer, err = getSigner(key)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported private key")
	}
	return marshalKeyPair(pemBlock.Type, Keys{signer, signer.Public()}, pass)
}

func getSigner(key any) (crypto.Signer, error) {
	var pk crypto.Signer
	switch k := key.(type) {
	case *rsa.PrivateKey:
		if err := cryptoutils.ValidatePubKey(k.Public()); err != nil {
			return nil, fmt.Errorf("error validating rsa key: %w", err)
		}
		pk = k
	case *ecdsa.PrivateKey:
		if err := cryptoutils.ValidatePubKey(k.Public()); err != nil {
			return nil, fmt.Errorf("error validating ecdsa key: %w", err)
		}
		pk = k
	case ed25519.PrivateKey:
		if err := cryptoutils.ValidatePubKey(k.Public()); err != nil {
			return nil, fmt.Errorf("error validating ed25519 key: %w", err)
		}
		pk = k
	case *ed25519.PrivateKey:
		return getSigner(*k)
	default:
		return nil, fmt.Errorf("unexpected private key type %T", k)
	}
	return pk, nil
}

func marshalKeyPair(ptype string, keypair Keys, pass []byte) (key *KeysBytes, err error) {
	x509Encoded, err := x509.MarshalPKCS8PrivateKey(keypair.private)
	if err != nil {
		return nil, fmt.Errorf("x509 encoding private key: %w", err)
	}

	encBytes, err := encrypted.Encrypt(x509Encoded, pass)
	if err != nil {
		return nil, err
	}

	// default to SIGSTORE, but keep support of COSIGN
	if ptype != cosign.CosignPrivateKeyPemType {
		ptype = cosign.SigstorePrivateKeyPemType
	}

	// store in PEM format
	privBytes := pem.EncodeToMemory(&pem.Block{
		Bytes: encBytes,
		Type:  ptype,
	})

	// Now do the public key
	pubBytes, err := cryptoutils.MarshalPublicKeyToPEM(keypair.public)
	if err != nil {
		return nil, err
	}

	return &KeysBytes{
		PrivateBytes: privBytes,
		PublicBytes:  pubBytes,
		password:     pass,
	}, nil
}
