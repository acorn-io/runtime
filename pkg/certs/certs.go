package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
)

const (
	CertificateBlockType   = "CERTIFICATE"
	ECPrivateKeyBlockType  = "EC PRIVATE KEY"
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"
)

func privKeyBlockType(algorithm v1.TLSAlgorithm) string {
	if algorithm == v1.TLSAlgorithmECDSA || algorithm == "" {
		return ECPrivateKeyBlockType
	}
	return RSAPrivateKeyBlockType
}

func toPEM(blockType string, data []byte) ([]byte, error) {
	pemBytes := &bytes.Buffer{}
	err := pem.Encode(pemBytes, &pem.Block{
		Type:  blockType,
		Bytes: data,
	})
	return pemBytes.Bytes(), err
}

func GenerateCA(algorithm v1.TLSAlgorithm) ([]byte, []byte, error) {
	pub, priv, privBytes, err := PubPrivateKey(algorithm)
	if err != nil {
		return nil, nil, err
	}

	before := time.Now().Add(-time.Hour)
	after := before.Add(365 * 24 * time.Hour)
	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("generated-ca@%d", time.Now().Unix()),
		},
		NotBefore: before,
		NotAfter:  after,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	cert, err := x509.CreateCertificate(cryptorand.Reader, &caTemplate, &caTemplate, pub, priv)
	if err != nil {
		return nil, nil, err
	}

	caCertPEM, err := toPEM(CertificateBlockType, cert)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := toPEM(privKeyBlockType(algorithm), privBytes)
	if err != nil {
		return nil, nil, err
	}

	return caCertPEM, keyPEM, nil
}

func ParseCert(data []byte) (*x509.Certificate, []byte, error) {
	block, _ := pem.Decode(data)
	if block.Type == CertificateBlockType {
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, err
		}
		return cert, block.Bytes, nil
	}
	return nil, nil, fmt.Errorf("invalid block type: %v", block.Type)
}

func ParsePubPrivateKey(data []byte) (any, any, []byte, error) {
	block, _ := pem.Decode(data)
	if block.Type == ECPrivateKeyBlockType {
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, nil, err
		}
		return &key.PublicKey, key, block.Bytes, nil
	} else if block.Type == RSAPrivateKeyBlockType {
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, nil, err
		}
		return &key.PublicKey, key, block.Bytes, nil
	}

	return nil, nil, nil, fmt.Errorf("unknown block type: %s", block.Type)
}

func GeneratePrivateKey(algorithm v1.TLSAlgorithm) ([]byte, error) {
	_, _, bytes, err := PubPrivateKey(algorithm)
	if err != nil {
		return nil, err
	}
	return toPEM(privKeyBlockType(algorithm), bytes)
}

func PubPrivateKey(algorithm v1.TLSAlgorithm) (any, any, []byte, error) {
	if algorithm == v1.TLSAlgorithmRSA {
		privKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
		if err != nil {
			return nil, nil, nil, err
		}
		bytes := x509.MarshalPKCS1PrivateKey(privKey)
		return &privKey.PublicKey, privKey, bytes, nil
	} else if algorithm == v1.TLSAlgorithmECDSA || algorithm == "" {
		privKey, err := ecdsa.GenerateKey(elliptic.P521(), cryptorand.Reader)
		if err != nil {
			return nil, nil, nil, err
		}
		bytes, err := x509.MarshalECPrivateKey(privKey)
		if err != nil {
			return nil, nil, nil, err
		}
		return &privKey.PublicKey, privKey, bytes, nil
	}
	return nil, nil, nil, fmt.Errorf("unknown algorithm %s", v1.TLSAlgorithmECDSA)
}

func GenerateCert(caCertPEM, caKeyPEM []byte, params v1.TLSParams) ([]byte, []byte, error) {
	var (
		validFrom = time.Now().Add(-time.Hour)
	)

	pub, _, privBytes, err := PubPrivateKey(params.Algorithm)
	if err != nil {
		return nil, nil, err
	}

	caBlock, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	_, caKey, _, err := ParsePubPrivateKey(caKeyPEM)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName:   params.CommonName,
			Organization: params.Organization,
		},
		NotBefore:             validFrom,
		NotAfter:              validFrom.Add(time.Hour * 24 * time.Duration(params.DurationDays)),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	switch params.Usage {
	case v1.CertUsageClient:
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	case v1.CertUsageServer:
		template.ExtKeyUsage = append(template.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	}

	for _, san := range params.SANs {
		ip := net.ParseIP(san)
		if ip == nil {
			template.DNSNames = append(template.DNSNames, san)
		} else {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}

	derBytes, err := x509.CreateCertificate(cryptorand.Reader, &template, caCert, pub, caKey)
	if err != nil {
		return nil, nil, err
	}

	certBytes, err := toPEM(CertificateBlockType, derBytes)
	if err != nil {
		return nil, nil, err
	}

	certBytes = append(certBytes, caCertPEM...)
	keyBytes, err := toPEM(privKeyBlockType(params.Algorithm), privBytes)
	if err != nil {
		return nil, nil, err
	}

	return certBytes, keyBytes, nil
}
