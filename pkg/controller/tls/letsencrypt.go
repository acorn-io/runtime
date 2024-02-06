package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/acorn-io/baaah/pkg/router"
	"github.com/acorn-io/runtime/pkg/config"
	"github.com/acorn-io/runtime/pkg/k8sclient"
	"github.com/acorn-io/runtime/pkg/labels"
	"github.com/acorn-io/runtime/pkg/system"
	"github.com/go-acme/lego/v4/certcrypto"
	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/dns01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LetsEncryptURLStaging    = "https://acme-staging-v02.api.letsencrypt.org/directory"
	LetsEncryptURLProduction = "https://acme-v02.api.letsencrypt.org/directory"
)

var (
	CertificatesRequestLock         = &sync.Mutex{}
	CertificateRequests             = map[string]any{}
	ErrCertificateRequestInProgress = errors.New("certificate request in progress")
)

type LEUser struct {
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
	if u.email == "" {
		return fmt.Errorf("not registering LE User: missing email")
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

func (u *LEUser) leClient() (*lego.Client, error) {
	conf := lego.NewConfig(u)

	conf.CADirURL = u.url

	client, err := lego.NewClient(conf)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func matchLeURLToEnv(url string) string {
	if url == LetsEncryptURLStaging {
		return "staging"
	} else if url == LetsEncryptURLProduction {
		return "enabled"
	}
	return "disabled"
}

// toHash returns a hash of the configurable fields of the LEUser
// It is used to determine if the LEUser has changed and needs to be updated.
// For this, we only check the email and url fields, since the key and registration are generated
// and identify the user against the ACME server.
func (u *LEUser) toHash() string {
	toHash := []byte(limit(fmt.Sprintf("%s-%s", matchLeURLToEnv(u.url), u.email), 63))
	dig := sha1.New()
	dig.Write(toHash)
	return hex.EncodeToString(dig.Sum(nil))
}

func limit(s string, count int) string {
	if len(s) < count {
		return s
	}
	return fmt.Sprintf("%s-%s", s[:count-6], toHex(s, 5))
}

func toHex(s string, length int) string {
	h := md5.Sum([]byte(s))
	d := hex.EncodeToString(h[:])
	return d[:length]
}

func ensureLEUser(ctx context.Context, client kclient.Client) (*LEUser, error) {
	cfg, err := config.Get(ctx, client)
	if err != nil {
		return nil, err
	}

	/*
	 * Construct new LE User
	 */
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
		email: email,
		url:   url,
	}

	newLEUserHash := newLEUser.toHash()

	/*
	 * Check for existing LE User in secret
	 */

	leAccountSecret := &corev1.Secret{}
	err = client.Get(ctx, router.Key(system.Namespace, system.LEAccountSecretName), leAccountSecret)
	if kclient.IgnoreNotFound(err) != nil {
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

	logrus.Infof("Registered Let's Encrypt User: %s", newLEUser.email)

	return newLEUser, nil
}

func (u *LEUser) certToSecret(cert *certificate.Resource, domain, namespace, name string) (*corev1.Secret, error) {
	leSettingsHash := u.toHash()

	x509crt, err := certcrypto.ParsePEMCertificate([]byte(cert.Certificate))
	if err != nil {
		return nil, fmt.Errorf("problem parsing pem certificate: %w", err)
	}

	sec := &corev1.Secret{
		Type: corev1.SecretTypeTLS,
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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

	return sec, nil
}

func (u *LEUser) getCert(ctx context.Context, domain string) (*certificate.Resource, error) {
	if strings.HasPrefix(domain, "*.") {
		return u.dnsChallenge(ctx, domain)
	}
	return nil, nil
}

func lockDomain(domain string) bool {
	CertificatesRequestLock.Lock()
	if _, ok := CertificateRequests[domain]; ok {
		CertificatesRequestLock.Unlock()
		logrus.Debugf("certificate for domain %s is already being requested, waiting for it to be ready", domain)
		return false
	}

	// Create "lock" for this domain
	CertificateRequests[domain] = nil
	CertificatesRequestLock.Unlock()
	return true
}

func unlockDomain(domain string) {
	CertificatesRequestLock.Lock()
	delete(CertificateRequests, domain)
	CertificatesRequestLock.Unlock()
}

// stillValid checks if the certificate is still valid for at least 7 days
func stillValid(cert []byte) bool {
	x509crt, err := certcrypto.ParsePEMCertificate(cert)
	if err != nil {
		// (a) unreadable certificate -> renew
		logrus.Errorf("problem parsing certificate: %v", err)
		return false
	}

	timeToExpire := x509crt.NotAfter.Sub(time.Now().UTC())
	if timeToExpire > 7*24*time.Hour {
		// (b) cert is still valid for more than 7 days -> good to go
		logrus.Debugf("certificate for %s is still valid until %s (%d hours)", x509crt.Subject.CommonName, x509crt.NotAfter, int(timeToExpire.Hours()))
		return true
	}
	// (c) cert is expired -> renew
	logrus.Infof("certificate for %s is expiring after %s (%d hours) and should be renewed", x509crt.Subject.CommonName, x509crt.NotAfter, int(timeToExpire.Hours()))
	return false
}

// mustRenew returns true if the certificate must be renewed, either because the Let's Encrypt settings changed, the certificate is invalid or it's about to expire
func (u *LEUser) mustRenew(sec *corev1.Secret) bool {
	// (a) let's encrypt user settings changed -> renew
	if sec.Annotations[labels.AcornLetsEncryptSettingsHash] != u.toHash() {
		logrus.Infof("let's encrypt settings changed, must renew certificate for %s", sec.Annotations[labels.AcornDomain])
		return true
	}

	// (b) certificate is expired or expiring soon or unreadable -> renew
	if !stillValid([]byte(sec.Data[corev1.TLSCertKey])) {
		return true
	}

	return false
}

func (u *LEUser) dnsChallenge(ctx context.Context, domain string) (*certificate.Resource, error) {
	client, err := u.leClient()
	if err != nil {
		return nil, err
	}

	c, err := k8sclient.Default()
	if err != nil {
		return nil, err
	}

	// Get DNS config
	cfg, err := config.Get(ctx, c)
	if err != nil {
		return nil, err
	}

	dnsEndpoint := *cfg.AcornDNSEndpoint

	dnsSecret := &corev1.Secret{}
	err = c.Get(ctx, router.Key(system.Namespace, system.DNSSecretName), dnsSecret)
	if kclient.IgnoreNotFound(err) != nil {
		return nil, err
	}
	token := string(dnsSecret.Data["token"])

	dnsProvider := NewACMEDNS01ChallengeProvider(dnsEndpoint, strings.TrimPrefix(domain, "*"), token)

	if err := client.Challenge.SetDNS01Provider(dnsProvider, dns01.WrapPreCheck(noOpCheck)); err != nil {
		return nil, err
	}

	// Try to obtain the certificate
	request := certificate.ObtainRequest{
		Domains: []string{strings.TrimPrefix(domain, ".")},
		Bundle:  true,
	}

	return client.Certificate.Obtain(request)
}
