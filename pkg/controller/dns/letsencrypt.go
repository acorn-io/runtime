package dns

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LEEndpoint = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

type LEUser struct {
	Name         string
	Email        string
	Registration *registration.Resource
	Key          crypto.PrivateKey
	URL          string
}

func FromSecret(secret *corev1.Secret) (*LEUser, error) {
	block, _ := pem.Decode(secret.Data["privateKey"])
	x509Encoded := block.Bytes
	privateKey, err := x509.ParseECPrivateKey(x509Encoded)
	if err != nil {
		return nil, err
	}

	var reg registration.Resource
	if err := json.Unmarshal(secret.Data["registration"], &reg); err != nil {
		return nil, err
	}

	return &LEUser{
		Name:         string(secret.Data["name"]),
		Email:        string(secret.Data["email"]),
		Registration: &reg,
		Key:          privateKey,
		URL:          string(secret.Data["url"]),
	}, nil
}

func (u *LEUser) GetEmail() string {
	return u.Email
}
func (u LEUser) GetRegistration() *registration.Resource {
	return u.Registration
}
func (u *LEUser) GetPrivateKey() crypto.PrivateKey {
	return u.Key
}

func (u *LEUser) Register() error {
	if u.Name == "" || u.Email == "" {
		return fmt.Errorf("not registering LE User: missing name or email")
	}

	if u.URL == "" {
		u.URL = LEEndpoint
	}

	if u.Key == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		u.Key = key
	}

	conf := lego.NewConfig(u)

	conf.CADirURL = u.URL

	client, err := lego.NewClient(conf)
	if err != nil {
		return err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	u.Registration = reg

	logrus.Infof("registered LE User: %s", u.Name)

	return nil

}

func (u *LEUser) ToSecret() (*corev1.Secret, error) {
	if u.Registration == nil {
		return nil, fmt.Errorf("not saving LE User: missing registration")
	}
	x509Encoded, _ := x509.MarshalECPrivateKey(u.Key.(*ecdsa.PrivateKey))
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	reg, err := json.Marshal(u.Registration)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.LESecretName,
			Namespace: system.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Data: map[string][]byte{
			"name":         []byte(u.Name),
			"email":        []byte(u.Email),
			"privateKey":   pemEncoded,
			"registration": reg,
			"url":          []byte(u.URL),
		},
	}, nil
}

func (u *LEUser) GenerateWildcardCert(dnsendpoint, domain, token string) (*certificate.Resource, error) {
	if u.Registration == nil {
		return nil, fmt.Errorf("not generating LE cert: missing registration")
	}

	if domain == "" || token == "" {
		return nil, fmt.Errorf("not generating LE cert: missing domain or token")
	}

	conf := lego.NewConfig(u)

	conf.CADirURL = u.URL

	client, err := lego.NewClient(conf)
	if err != nil {
		return nil, err
	}

	dnsProvider := NewDNSProvider(dnsendpoint, domain, token)

	if err := client.Challenge.SetDNS01Provider(dnsProvider); err != nil {
		return nil, err
	}

	request := certificate.ObtainRequest{
		Domains: []string{fmt.Sprintf("*.%s", strings.TrimPrefix(domain, "."))},
		Bundle:  true,
	}

	return client.Certificate.Obtain(request)
}

func EnsureLEUser(ctx context.Context, client kclient.Client, domain string) (*LEUser, error) {

	targetUsername := strings.TrimPrefix(domain, ".") // leading dot is an issue especially for email addresses

	leAccountSecret := &corev1.Secret{}
	err := client.Get(ctx, router.Key(system.Namespace, system.LESecretName), leAccountSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	// Existing LE User Secret found
	if err == nil {
		leUser, err := FromSecret(leAccountSecret)
		if err != nil {
			return nil, err
		}

		// Domain changed, recreate the secret
		if !strings.Contains(leUser.Name, targetUsername) {
			logrus.Infof("deleting LE secret for domain %s", domain)
			if err := client.Delete(ctx, leAccountSecret); err != nil {
				return nil, err
			}
		} else {
			return leUser, nil
		}
	}

	// Create and Register Let's Encrypt User
	leUser := &LEUser{
		Name:  targetUsername,
		Email: fmt.Sprintf("%s@on-acorn.io", targetUsername),
		URL:   LEEndpoint,
	}
	if err := leUser.Register(); err != nil {
		return nil, fmt.Errorf("problem registering Let's Encrypt User: %w", err)
	}

	sec, err := leUser.ToSecret()
	if err != nil {
		return nil, fmt.Errorf("problem creating Let's Encrypt User secret: %w", err)
	}

	if err := client.Create(ctx, sec); err != nil {
		return nil, fmt.Errorf("problem creating Let's Encrypt User secret: %w", err)
	}

	logrus.Infof("Registered Let's Encrypt User: %s", leUser.Name)

	return leUser, nil

}

func (u *LEUser) EnsureWildcardCertificateSecret(ctx context.Context, client kclient.Client, dnsendpoint, domain, token string) (*corev1.Secret, error) {
	cert, err := u.GenerateWildcardCert(dnsendpoint, domain, token)
	if err != nil {
		return nil, fmt.Errorf("problem generating wildcard certificate: %w", err)
	}

	sec := &corev1.Secret{
		Type: corev1.SecretTypeTLS,
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.TLSSecretName,
			Namespace: system.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Data: map[string][]byte{
			"tls.crt": cert.Certificate,
			"tls.key": cert.PrivateKey,
		},
	}

	if err := client.Create(ctx, sec); err != nil {
		return sec, fmt.Errorf("problem creating wildcard certificate secret: %w", err)
	}

	logrus.Infof("Created wildcard certificate secret for domain %s", domain)

	return sec, nil
}
