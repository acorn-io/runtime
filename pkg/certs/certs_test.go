package certs

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"

	v1 "github.com/acorn-io/acorn/pkg/apis/internal.acorn.io/v1"
	"github.com/stretchr/testify/assert"
)

func TestRSA_CA(t *testing.T) {
	cert, caKey, err := GenerateCA(v1.TLSAlgorithmRSA)
	if err != nil {
		t.Fatal(err)
	}

	x509Cert, _, err := ParseCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(caKey)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 256, privateKey.PublicKey.Size())
	assert.True(t, x509Cert.IsCA)
	assert.Equal(t, "RSA", x509Cert.PublicKeyAlgorithm.String())
	assert.True(t, strings.HasPrefix(x509Cert.Subject.CommonName, "generated-ca@"))
}

func TestECDSA_CA(t *testing.T) {
	cert, caKey, err := GenerateCA(v1.TLSAlgorithmECDSA)
	if err != nil {
		t.Fatal(err)
	}

	x509Cert, _, err := ParseCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(caKey)
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 521, privateKey.Curve.Params().BitSize)
	assert.True(t, x509Cert.IsCA)
	assert.Equal(t, "ECDSA", x509Cert.PublicKeyAlgorithm.String())
	assert.True(t, strings.HasPrefix(x509Cert.Subject.CommonName, "generated-ca@"))
}

func TestDefault_Server_Cert(t *testing.T) {
	caCert, caKey, err := GenerateCA("")
	if err != nil {
		t.Fatal(err)
	}

	cert, key, err := GenerateCert(caCert, caKey, v1.TLSParams{
		Algorithm:    "",
		CASecret:     "",
		Usage:        "server",
		CommonName:   "cn",
		Organization: []string{"org1", "org2"},
		SANs:         []string{"host1", "127.0.0.1", "host2"},
		DurationDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	x509Cert, _, err := ParseCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(key)
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 521, privateKey.Curve.Params().BitSize)
	assert.False(t, x509Cert.IsCA)
	assert.Equal(t, "ECDSA", x509Cert.PublicKeyAlgorithm.String())
	assert.Equal(t, "cn", x509Cert.Subject.CommonName)
	assert.Equal(t, []string{"host1", "host2"}, x509Cert.DNSNames)
	assert.Equal(t, 1, len(x509Cert.IPAddresses))
	assert.Equal(t, "127.0.0.1", x509Cert.IPAddresses[0].String())
	assert.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, x509Cert.ExtKeyUsage)
}

func TestRSA_Server_Cert(t *testing.T) {
	caCert, caKey, err := GenerateCA(v1.TLSAlgorithmRSA)
	if err != nil {
		t.Fatal(err)
	}

	cert, key, err := GenerateCert(caCert, caKey, v1.TLSParams{
		Algorithm:    v1.TLSAlgorithmRSA,
		CASecret:     "",
		Usage:        "server",
		CommonName:   "cn",
		Organization: []string{"org1", "org2"},
		SANs:         []string{"host1", "127.0.0.1", "host2"},
		DurationDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	x509Cert, _, err := ParseCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(key)
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 256, privateKey.PublicKey.Size())
	assert.False(t, x509Cert.IsCA)
	assert.Equal(t, "RSA", x509Cert.PublicKeyAlgorithm.String())
	assert.Equal(t, "cn", x509Cert.Subject.CommonName)
	assert.Equal(t, []string{"host1", "host2"}, x509Cert.DNSNames)
	assert.Equal(t, 1, len(x509Cert.IPAddresses))
	assert.Equal(t, "127.0.0.1", x509Cert.IPAddresses[0].String())
	assert.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, x509Cert.ExtKeyUsage)
}

func TestECDSA_Client_Cert(t *testing.T) {
	caCert, caKey, err := GenerateCA(v1.TLSAlgorithmECDSA)
	if err != nil {
		t.Fatal(err)
	}

	cert, key, err := GenerateCert(caCert, caKey, v1.TLSParams{
		Algorithm:    v1.TLSAlgorithmECDSA,
		CASecret:     "",
		Usage:        "client",
		CommonName:   "cn",
		Organization: []string{"org1", "org2"},
		SANs:         []string{"host1", "127.0.0.1", "host2"},
		DurationDays: 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	x509Cert, _, err := ParseCert(cert)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(key)
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 521, privateKey.Curve.Params().BitSize)
	assert.False(t, x509Cert.IsCA)
	assert.Equal(t, "ECDSA", x509Cert.PublicKeyAlgorithm.String())
	assert.Equal(t, "cn", x509Cert.Subject.CommonName)
	assert.Equal(t, []string{"host1", "host2"}, x509Cert.DNSNames)
	assert.Equal(t, 1, len(x509Cert.IPAddresses))
	assert.Equal(t, "127.0.0.1", x509Cert.IPAddresses[0].String())
	assert.Equal(t, []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, x509Cert.ExtKeyUsage)
}
