package dns

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	apiv1 "github.com/acorn-io/acorn/pkg/apis/api.acorn.io/v1"
	"github.com/acorn-io/acorn/pkg/labels"
	"github.com/acorn-io/acorn/pkg/system"
	"github.com/acorn-io/baaah/pkg/router"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LetsEncryptURLStaging    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptURLProduction = "https://acme-v02.api.letsencrypt.org/directory"
)

type LEUser struct {
	name         string
	email        string
	registration *registration.Resource
	key          crypto.PrivateKey
	url          string
}

func fromSecret(secret *corev1.Secret) (*LEUser, error) {
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
		name:         string(secret.Data["name"]),
		email:        string(secret.Data["email"]),
		registration: &reg,
		key:          privateKey,
		url:          string(secret.Data["url"]),
	}, nil
}

func (u *LEUser) GetEmail() string {
	return u.email
}
func (u *LEUser) GetRegistration() *registration.Resource {
	return u.registration
}
func (u *LEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

func (u *LEUser) register() error {
	if u.name == "" || u.email == "" {
		return fmt.Errorf("not registering LE User: missing name or email")
	}

	if u.url == "" {
		return fmt.Errorf("not registering LE User: missing URL")
	}

	if u.key == nil {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		u.key = key
	}

	conf := lego.NewConfig(u)

	conf.CADirURL = u.url

	client, err := lego.NewClient(conf)
	if err != nil {
		return err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return err
	}
	u.registration = reg

	logrus.Infof("registered LE User: %s", u.name)

	return nil

}

func (u *LEUser) toSecret() (*corev1.Secret, error) {
	if u.registration == nil {
		return nil, fmt.Errorf("not saving LE User: missing registration")
	}
	x509Encoded, _ := x509.MarshalECPrivateKey(u.key.(*ecdsa.PrivateKey))
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	reg, err := json.Marshal(u.registration)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.LEAccountSecretName,
			Namespace: system.Namespace,
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Data: map[string][]byte{
			"name":         []byte(u.name),
			"email":        []byte(u.email),
			"privateKey":   pemEncoded,
			"registration": reg,
			"url":          []byte(u.url),
		},
	}, nil
}

func noOpCheck(_, _, _ string, _ dns01.PreCheckFunc) (bool, error) {
	return true, nil
}

func (u *LEUser) generateWildcardCert(dnsendpoint, domain, token string) (*certificate.Resource, error) {
	if u.registration == nil {
		return nil, fmt.Errorf("not generating LE cert: missing registration")
	}

	if domain == "" || token == "" {
		return nil, fmt.Errorf("not generating LE cert: missing domain or token")
	}

	conf := lego.NewConfig(u)

	conf.CADirURL = u.url

	client, err := lego.NewClient(conf)
	if err != nil {
		return nil, err
	}

	dnsProvider := NewACMEDNS01ChallengeProvider(dnsendpoint, domain, token)

	if err := client.Challenge.SetDNS01Provider(dnsProvider, dns01.WrapPreCheck(noOpCheck)); err != nil {
		return nil, err
	}

	request := certificate.ObtainRequest{
		Domains: []string{fmt.Sprintf("*.%s", strings.TrimPrefix(domain, "."))},
		Bundle:  true,
	}

	return client.Certificate.Obtain(request)
}

func matchLeURLToEnv(url string) string {
	if url == LetsEncryptURLStaging {
		return "staging"
	} else if url == LetsEncryptURLProduction {
		return "enabled"
	} else {
		return "disabled"
	}
}

func (u *LEUser) toHash() string {
	toHash := []byte(fmt.Sprintf("%s-%s-%s", u.name, matchLeURLToEnv(u.url), u.email))
	dig := sha1.New()
	dig.Write([]byte(toHash))
	return hex.EncodeToString(dig.Sum(nil))
}

func ensureLEUser(ctx context.Context, cfg *apiv1.Config, client kclient.Client, domain string) (*LEUser, error) {

	/*
	 * Construct new LE User
	 */
	targetUsername := strings.TrimPrefix(domain, ".") // leading dot can be an issue
	email := "staging-certs@acorn.io"
	url := LetsEncryptURLStaging
	if strings.EqualFold(*cfg.LetsEncrypt, "enabled") {
		url = LetsEncryptURLProduction
		if cfg.LetsEncryptEmail == "" {
			return nil, fmt.Errorf("let's encrypt email is required")
		}
	}
	if cfg.LetsEncryptEmail != "" {
		email = cfg.LetsEncryptEmail
	}

	newLEUser := &LEUser{
		name:  targetUsername,
		email: email,
		url:   url,
	}

	newLEUserHash := newLEUser.toHash()

	/*
	 * Check for existing LE User in secret
	 */

	leAccountSecret := &corev1.Secret{}
	err := client.Get(ctx, router.Key(system.Namespace, system.LEAccountSecretName), leAccountSecret)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	// Existing LE User Secret found
	if err == nil {
		currentLEUser, err := fromSecret(leAccountSecret)
		if err != nil {
			return nil, err
		}

		currentLEUserHash := currentLEUser.toHash()

		// Domain, LE environment or LE email changed -> delete secret for re-creation
		if currentLEUserHash != newLEUserHash {
			logrus.Infof("deleting let's encrypt secret due to config change: %v -> %v", currentLEUser, newLEUser)
			err = client.Delete(ctx, leAccountSecret)
			if err != nil {
				return nil, err
			}
		} else {
			return currentLEUser, nil
		}
	}

	if err := newLEUser.register(); err != nil {
		return nil, fmt.Errorf("problem registering Let's Encrypt User: %w", err)
	}

	sec, err := newLEUser.toSecret()
	if err != nil {
		return nil, fmt.Errorf("problem creating Let's Encrypt User secret: %w", err)
	}

	if err := client.Create(ctx, sec); err != nil {
		return nil, fmt.Errorf("problem creating Let's Encrypt User secret: %w", err)
	}

	logrus.Infof("Registered Let's Encrypt User: %s", newLEUser.name)

	return newLEUser, nil

}

func (u *LEUser) ensureWildcardCertificateSecret(ctx context.Context, client kclient.Client, dnsendpoint, domain, token string) (*corev1.Secret, error) {

	sec := &corev1.Secret{}
	secErr := client.Get(ctx, router.Key(system.Namespace, system.TLSSecretName), sec)
	if secErr != nil && !apierrors.IsNotFound(secErr) {
		// error fetching the existing secret, but it could exist
		return nil, secErr
	}

	leSettingsHash := u.toHash()

	// Existing LE Cert Secret found, let's check if it's still valid
	if secErr == nil {

		// (a) domain or let's encrypt user settings changed -> renew
		if sec.Annotations[labels.AcornLetsEncryptSettingsHash] != leSettingsHash {
			logrus.Info("domain or let's encrypt settings changed, renewing wildcard certificate")
		} else {

			x509crt, err := certcrypto.ParsePEMCertificate([]byte(sec.Data[corev1.TLSCertKey]))
			if err != nil {
				// (b) unreadable certificate -> renew
				logrus.Errorf("problem parsing existing TLS secret: %v", err)
			} else {
				timeToExpire := x509crt.NotAfter.Sub(time.Now().UTC())
				if timeToExpire > 7*24*time.Hour {
					// (c) cert is still valid for more than 7 days -> return it
					logrus.Infof("existing TLS secret %s is still valid until %s (%d hours)", x509crt.Subject.CommonName, x509crt.NotAfter, int(timeToExpire.Hours()))
					return sec, nil
				} else {
					// (d) cert is expired -> renew
					logrus.Infof("existing TLS secret %s is expiring after %s (%d hours), renewing it...", x509crt.Subject.CommonName, x509crt.NotAfter, int(timeToExpire.Hours()))
				}
			}
		}
	}

	cert, err := u.generateWildcardCert(dnsendpoint, domain, token)
	if err != nil {
		return nil, fmt.Errorf("problem generating wildcard certificate: %w", err)
	}

	x509crt, err := certcrypto.ParsePEMCertificate([]byte(cert.Certificate))
	if err != nil {
		return nil, fmt.Errorf("problem parsing pem certificate: %w", err)
	}

	sec = &corev1.Secret{
		Type: corev1.SecretTypeTLS,
		ObjectMeta: metav1.ObjectMeta{
			Name:      system.TLSSecretName,
			Namespace: system.Namespace,
			Annotations: map[string]string{
				labels.AcornDomain:                  domain,
				labels.AcornLetsEncryptSettingsHash: leSettingsHash,
				labels.AcornCertNotValidBefore:      x509crt.NotBefore.Format(time.RFC3339),
				labels.AcornCertNotValidAfter:       x509crt.NotAfter.Format(time.RFC3339),
			},
			Labels: map[string]string{
				labels.AcornManaged: "true",
			},
		},
		Data: map[string][]byte{
			corev1.TLSCertKey:       cert.Certificate,
			corev1.TLSPrivateKeyKey: cert.PrivateKey,
		},
	}

	if apierrors.IsNotFound(secErr) {
		if err := client.Create(ctx, sec); err != nil {
			return sec, fmt.Errorf("problem creating wildcard certificate secret: %w", err)
		}
	} else {
		if err := client.Update(ctx, sec); err != nil {
			return sec, fmt.Errorf("problem updating wildcard certificate secret: %w", err)
		}
	}

	logrus.Infof("Created new wildcard certificate secret for domain %s", domain)

	return sec, nil
}
